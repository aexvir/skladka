package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/paste"
	"github.com/aexvir/skladka/internal/storage/sql"
	"github.com/aexvir/skladka/internal/tracing"
)

type PostgresStorage struct {
	conn    *pgxpool.Pool
	db      *sql.Queries
	cipher  *Cipher
	metrics *Metrics
}

type PostgresStorageOption func(*PostgresStorage)

func NewPostgresStorage(ctx context.Context, cfg config.Config, opts ...PostgresStorageOption) (*PostgresStorage, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "storage.NewPostgresStorage")
	defer finish(&err)

	logging.
		FromContext(ctx).
		Info("storage.postgres", "initializing postgres storage", "url", cfg.Postgres.URL)

	connstr := cfg.Postgres.URL

	if cfg.Postgres.URL == "" {
		connstr = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s",
			cfg.Postgres.User, cfg.Postgres.Password,
			cfg.Postgres.Host, cfg.Postgres.Port,
			cfg.Postgres.Database,
		)
	}

	conn, err := pgxpool.New(ctx, connstr)
	if err != nil {
		return nil, err
	}

	met := new(Metrics)
	if err := metrics.FromContext(ctx).Register(met); err != nil {
		return nil, errors.Wrap(err, "registering metrics")
	}

	store := PostgresStorage{
		conn:    conn,
		db:      sql.New(conn),
		metrics: met,
		cipher:  NewCipher(cfg.EncryptionKey, cfg.EncryptionSalt),
	}

	for _, opt := range opts {
		opt(&store)
	}

	return &store, nil
}

func (s *PostgresStorage) CreatePaste(ctx context.Context, paste paste.Paste) (string, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.CreatePaste")
	defer finish(&err)

	ref, err := s.ref(10)
	if err != nil {
		s.metrics.PasteErrors.Add(ctx, 1)
		return "", err
	}

	if paste.Password != nil {
		hash := s.cipher.Hash(*paste.Password)
		paste.Password = &hash
	}

	if err := s.EncryptPaste(&paste); err != nil {
		return "", errors.Wrap(err, "failed to encrypt data")
	}

	row := new(sql.Paste).FromDomain(paste)

	_, err = s.db.CreatePaste(
		ctx, sql.CreatePasteParams{
			Reference:  ref,
			Title:      row.Title,
			Content:    row.Content,
			Syntax:     row.Syntax,
			Tags:       row.Tags,
			Expiration: row.Expiration,
			Public:     row.Public,
			Password:   row.Password,
		},
	)

	if err != nil {
		s.metrics.PasteErrors.Add(ctx, 1)
		return "", err
	}

	s.metrics.PasteCreated.Add(ctx, 1)
	s.metrics.PasteSize.Record(ctx, int64(len(row.Content)))

	return ref, nil
}

func (s *PostgresStorage) GetPaste(ctx context.Context, ref string) (paste.Paste, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.GetPaste")
	defer finish(&err)

	var empty paste.Paste

	row, err := s.db.GetPasteByReference(ctx, ref)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.metrics.PasteNotFound.Add(ctx, 1)
		} else {
			s.metrics.PasteErrors.Add(ctx, 1)
		}
		return empty, err
	}

	paste := row.ToDomain()
	if err := s.DecryptPaste(&paste); err != nil {
		return empty, errors.Wrap(err, "failed to decrypt data")
	}

	s.metrics.PasteRetrieved.Add(ctx, 1)

	return paste, nil
}

func (s *PostgresStorage) GetPasteWithPassword(ctx context.Context, ref, password string) (*paste.Paste, error) {
	paste, err := s.GetPaste(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get paste")
	}

	if paste.Password == nil {
		return nil, errors.Errorf("paste %s doesn't have a password", ref)
	}

	if !s.cipher.Verify(password, *paste.Password) {
		return nil, nil
	}

	return &paste, nil
}

func (s *PostgresStorage) ListPastes(ctx context.Context) ([]paste.Paste, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "storage.ListPastes")
	defer finish(&err)

	logger := logging.FromContext(ctx)
	logger.Info("storage.postgres", "listing public pastes")

	dbPastes, err := s.db.ListPublicPastes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list public pastes")
	}

	pastes := make([]paste.Paste, len(dbPastes))
	for i, row := range dbPastes {
		paste := row.ToDomain()

		if err := s.DecryptPaste(&paste); err != nil {
			continue
		}

		pastes[i] = paste
	}

	return pastes, nil
}

func (s *PostgresStorage) EncryptPaste(paste *paste.Paste) error {
	var errt, errc error
	paste.Title, errt = s.cipher.Encrypt(paste.Title)
	paste.Content, errc = s.cipher.Encrypt(paste.Content)

	if errt != nil || errc != nil {
		return errors.Join(errt, errc)
	}

	return nil
}

func (s *PostgresStorage) DecryptPaste(paste *paste.Paste) error {
	var errt, errc error
	paste.Title, errt = s.cipher.Decrypt(paste.Title)
	paste.Content, errc = s.cipher.Decrypt(paste.Content)

	if errt != nil || errc != nil {
		return errors.Join(errt, errc)
	}

	return nil
}

func (s *PostgresStorage) ref(attempts int) (string, error) {
	attempt := 0

	for {
		if attempt >= attempts {
			return "", errors.Errorf("failed to generate unique ref in %d attempts", attempt)
		}

		attempt++

		ref, err := generateReferenceIdentifier()
		if err == nil {
			return ref, nil
		}
	}
}
