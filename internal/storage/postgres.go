package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/aexvir/skladka/internal/auth"
	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/paste"
	"github.com/aexvir/skladka/internal/storage/sql"
	"github.com/aexvir/skladka/internal/tracing"
)

type PostgresStorage struct {
	conn   *pgxpool.Pool
	db     *sql.Queries
	cipher *Cipher
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

	store := PostgresStorage{
		conn:   conn,
		db:     sql.New(conn),
		cipher: NewCipher(cfg.EncryptionKey, cfg.EncryptionSalt),
	}

	for _, opt := range opts {
		opt(&store)
	}

	go store.expiration(ctx, 5*time.Second)

	return &store, nil
}

// CreateUser creates a new user in the database with the provided [auth.User] data.
func (s *PostgresStorage) CreateUser(ctx context.Context, user auth.User) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.CreateUser")
	defer finish(&err)

	row := new(sql.User).FromDomain(user)

	_, err = s.db.CreateUser(
		ctx,
		sql.CreateUserParams{
			Username:    row.Username,
			Uuid:        row.Uuid,
			Credentials: row.Credentials,
		},
	)

	return err
}

// GetUserByUsername retrieves a user from the database by their username.
// It returns the user data as an [auth.User] object and any error that occurred during the operation.
func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (user auth.User, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.GetUserByUsername")
	defer finish(&err)

	row, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return user, err
	}

	return row.ToDomain(), nil
}

// UpdateUserCredentials updates the stored credentials for a user in the database.
// It expects the [auth.User] object to have the new value for Credentials already set.
// Returns an error if the update operation fails.
func (s *PostgresStorage) UpdateUserCredentials(ctx context.Context, user auth.User) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.UpdateUserCredentials")
	defer finish(&err)

	row := new(sql.User).FromDomain(user)

	return s.db.UpdateUserCredentials(
		ctx,
		sql.UpdateUserCredentialsParams{
			Username:    row.Username,
			Credentials: row.Credentials,
		},
	)
}

// UpdateUserAvatar updates the avatar image data for a user in the database.
// Returns an error if the database update operation fails.
func (s *PostgresStorage) UpdateUserAvatar(ctx context.Context, user auth.User) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.UpdateUserAvatar")
	defer finish(&err)

	row := new(sql.User).FromDomain(user)

	return s.db.UpdateUserAvatar(
		ctx,
		sql.UpdateUserAvatarParams{
			Username: row.Username,
			Avatar:   row.Avatar,
		},
	)
}

// CreateSession creates a new user session in the database with the provided [auth.Session] data.
// Returns an error if the database operation fails.
func (s *PostgresStorage) CreateSession(ctx context.Context, ssn auth.Session) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.CreateSession")
	defer finish(&err)

	row := new(sql.Session).FromDomain(ssn)

	_, err = s.db.CreateSession(
		ctx,
		sql.CreateSessionParams{
			Token:     row.Token,
			Username:  row.Username,
			Data:      row.Data,
			ExpiresAt: row.ExpiresAt,
			Throwaway: row.Throwaway,
		},
	)

	return err
}

// GetSessionByToken retrieves a session from the database using the provided token string.
// Returns the session data as an [auth.Session] object if found, or an error if the token
// is invalid or the session cannot be retrieved.
func (s *PostgresStorage) GetSessionByToken(ctx context.Context, token string) (session auth.Session, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.GetSessionByToken")
	defer finish(&err)

	var uuid pgtype.UUID
	if err := uuid.Scan(token); err != nil {
		return session, errors.Wrap(err, "failed to convert session uuid to pg type")
	}

	row, err := s.db.GetSessionByToken(ctx, uuid)
	if err != nil {
		return session, err
	}

	return row.ToDomain(), nil
}

// CreatePaste stores a new paste in the database. It generates a unique reference identifier,
// encrypts the paste content and title if needed, and hashes any provided password.
//
// Returns:
//   - reference: The unique reference string used to identify the paste
//   - err: Any error that occurred during the operation
//
// The method will attempt to generate a unique reference up to 10 times before failing.
func (s *PostgresStorage) CreatePaste(ctx context.Context, paste paste.Paste) (reference string, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.CreatePaste")
	defer finish(&err)

	ref, err := s.ref(10)
	if err != nil {
		return "", err
	}

	if paste.Password != nil {
		hash := s.cipher.Hash(*paste.Password)
		paste.Password = &hash
	}

	if err := s.EncryptPaste(ctx, &paste); err != nil {
		return "", errors.Wrap(err, "failed to encrypt data")
	}

	row := new(sql.Paste).FromDomain(paste)

	_, err = s.db.CreatePaste(
		ctx, sql.CreatePasteParams{
			Reference:  ref,
			Owner:      row.Owner,
			Title:      row.Title,
			Content:    row.Content,
			Mimetype:   row.Mimetype,
			Syntax:     row.Syntax,
			Tags:       row.Tags,
			Expiration: row.Expiration,
			Public:     row.Public,
			Password:   row.Password,
		},
	)
	if err != nil {
		return "", err
	}

	return ref, nil
}

// GetPaste retrieves a paste from the database using its reference identifier.
// It fetches the encrypted paste data and decrypts it before returning.
//
// Returns:
//   - paste.Paste: The decrypted paste if found
//   - error: Any error that occurred during fetching or decryption
//
// If the paste cannot be found or decrypted, returns an empty paste and the error.
func (s *PostgresStorage) GetPaste(ctx context.Context, ref string) (paste paste.Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.GetPaste")
	defer finish(&err)

	row, err := s.db.GetPasteByReference(ctx, ref)
	if err != nil {
		return paste, err
	}

	paste = row.ToDomain()
	if err := s.DecryptPaste(ctx, &paste); err != nil {
		return paste, errors.Wrap(err, "failed to decrypt data")
	}

	return paste, nil
}

// GetPasteWithPassword retrieves a password-protected paste from storage and verifies the provided password.
// It first fetches the paste using the reference identifier, then checks if it's password protected
// and verifies the supplied password matches.
//
// Returns:
//   - *paste.Paste: The paste if found and password verified, nil if password invalid
//   - error: Error if paste not found, has no password, or other errors occur
func (s *PostgresStorage) GetPasteWithPassword(ctx context.Context, ref, password string) (*paste.Paste, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.GetPasteWithPassword")
	defer finish(&err)

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

func (s *PostgresStorage) DeletePaste(ctx context.Context, ref string) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.DeletePaste")
	defer finish(&err)

	return s.db.DeletePaste(ctx, ref)
}

// ListPastes retrieves all public pastes from the database and decrypts their contents.
//
// This method:
// 1. Fetches all public pastes from the database
// 2. Converts each database row to domain model
// 3. Attempts to decrypt the content and title of each paste
// 4. Skips any pastes that fail decryption
//
// Returns:
// - []paste.Paste: Slice containing all successfully retrieved and decrypted public pastes
// - error: Any error encountered while fetching pastes from the database
func (s *PostgresStorage) ListPastes(ctx context.Context) (pastes []paste.Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.ListPastes")
	defer finish(&err)

	logger := logging.FromContext(ctx)
	logger.Info("storage.postgres", "listing public pastes")

	rows, err := s.db.ListPublicPastes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list public pastes")
	}

	pastes = make([]paste.Paste, len(rows))
	for i, row := range rows {
		paste := row.ToDomain()

		if err := s.DecryptPaste(ctx, &paste); err != nil {
			continue
		}

		pastes[i] = paste
	}

	return pastes, nil
}

// ListUserPastes retrieves all pastes from the database that belong to the specified user.
// It fetches the encrypted paste data and decrypts it before returning.
//
// Returns:
//   - []paste.Paste: The decrypted pastes if found
//   - error: Any error that occurred during fetching or decryption
func (s *PostgresStorage) ListUserPastes(ctx context.Context, username string) (pastes []paste.Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.ListUserPastes")
	defer finish(&err)

	logger := logging.FromContext(ctx)
	logger.Info("storage.postgres", "listing user pastes", "username", username)

	rows, err := s.db.ListUserPastes(ctx, pgtype.Text{String: username, Valid: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user pastes")
	}

	pastes = make([]paste.Paste, len(rows))
	for i, row := range rows {
		paste := row.ToDomain()

		if err := s.DecryptPaste(ctx, &paste); err != nil {
			continue
		}

		pastes[i] = paste
	}

	return pastes, nil
}

// EncryptPaste encrypts both the title and content of a paste using the storage's cipher.
// The encryption is done in-place, modifying the provided paste object directly.
//
// The method will attempt to encrypt both fields even if one fails, then return any errors
// that occurred during either operation.
func (s *PostgresStorage) EncryptPaste(ctx context.Context, paste *paste.Paste) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.EncryptPaste")
	defer finish(&err)

	var errt, errc error
	paste.Title, errt = s.cipher.Encrypt(paste.Title)
	paste.Content, errc = s.cipher.Encrypt(paste.Content)

	if errt != nil || errc != nil {
		return errors.Join(errt, errc)
	}

	return nil
}

// DecryptPaste decrypts both the title and content of a paste using the storage's cipher.
// The decryption is done in-place, modifying the provided paste object directly.
//
// The method will attempt to decrypt both fields even if one fails, then return any errors
// that occurred during either operation joined together.
func (s *PostgresStorage) DecryptPaste(ctx context.Context, paste *paste.Paste) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "PostgresStorage.DecryptPaste")
	defer finish(&err)

	var errt, errc error
	paste.Title, errt = s.cipher.Decrypt(paste.Title)
	paste.Content, errc = s.cipher.Decrypt(paste.Content)

	if errt != nil || errc != nil {
		return errors.Join(errt, errc)
	}

	return nil
}

// expiration coroutine that runs every [interval].
// deletes expired pastes on every tick and reacts to context cancellation.
func (s *PostgresStorage) expiration(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	logger := logging.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("storage.postgres.expiration", "expiration coroutine stopped: context closed", "err", ctx.Err())
			return

		case <-ticker.C:
			start := time.Now()

			deleted, err := s.db.DeleteExpiredPastes(ctx)
			if err != nil {
				logger.Error(err, "storage.postgres.expiration", "failed to run expiration coroutine")
				continue
			}

			logger.Info(
				"storage.postgres.expiration",
				"expiration coroutine",
				attribute.Int("expired.count", len(deleted)),
				attribute.StringSlice("expired.refs", deleted),
				attribute.Int64("duration", time.Since(start).Milliseconds()),
			)
		}
	}
}

// ref attempts to generate a unique reference identifier by making multiple attempts.
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
