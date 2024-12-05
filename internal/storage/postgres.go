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

	row := new(sql.Paste).FromDomain(paste)
	if s.cipher != nil {
		var errt, errc error

		row.Title, errt = s.cipher.Encrypt(row.Title)
		row.Content, errc = s.cipher.Encrypt(row.Content)

		if errt != nil || errc != nil {
			s.metrics.PasteErrors.Add(ctx, 1)
			return "", errors.New("failed to encrypt data")
		}
	}

	_, err = s.db.CreatePaste(
		ctx, sql.CreatePasteParams{
			Reference:  ref,
			Title:      row.Title,
			Content:    row.Content,
			Syntax:     row.Syntax,
			Tags:       row.Tags,
			Expiration: row.Expiration,
			Public:     row.Public,
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

	if s.cipher != nil {
		var errt, errc error

		row.Title, errt = s.cipher.Decrypt(row.Title)
		row.Content, errc = s.cipher.Decrypt(row.Content)

		if errt != nil || errc != nil {
			s.metrics.PasteErrors.Add(ctx, 1)
			return empty, errors.New("failed to encrypt data")
		}
	}

	s.metrics.PasteRetrieved.Add(ctx, 1)
	return row.ToDomain(), nil
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
	for i, p := range dbPastes {
		pastes[i] = p.ToDomain()
	}

	return pastes, nil
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
