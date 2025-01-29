package sql

import (
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aexvir/skladka/internal/auth"
	"github.com/aexvir/skladka/internal/paste"
)

func (db User) ToDomain() auth.User {
	var credentials []webauthn.Credential
	if err := json.Unmarshal(db.Credentials, &credentials); err != nil {
		panic(err)
	}

	return auth.User{
		Username:    db.Username,
		UUID:        db.Uuid.String(),
		Credentials: credentials,
	}
}

func (User) FromDomain(domain auth.User) User {
	var uuid pgtype.UUID
	if err := uuid.Scan(domain.UUID); err != nil {
		panic(err)
	}

	credentials, err := json.Marshal(domain.Credentials)
	if err != nil {
		panic(err)
	}

	return User{
		Username:    domain.Username,
		Uuid:        uuid,
		Credentials: credentials,
	}
}

func (db Session) ToDomain() auth.Session {
	var expiration *time.Time
	if db.ExpiresAt.Valid {
		expiration = &db.ExpiresAt.Time
	}

	return auth.Session{
		Token:     db.Token.String(),
		Username:  db.Username,
		Data:      db.Data,
		ExpiresAt: *expiration,
	}
}

func (Session) FromDomain(domain auth.Session) Session {
	var token pgtype.UUID
	if err := token.Scan(domain.Token); err != nil {
		panic(err)
	}

	return Session{
		Username: domain.Username,
		Token:    token,
		Data:     domain.Data,
		ExpiresAt: pgtype.Timestamp{
			Time:  domain.ExpiresAt,
			Valid: true,
		},
	}
}

func (db Paste) ToDomain() paste.Paste {
	var owner *string
	if db.Owner.Valid {
		owner = &db.Owner.String
	}

	syntax := "plaintext"
	if db.Syntax.Valid {
		syntax = db.Syntax.String
	}

	var expiration *time.Time
	if db.Expiration.Valid {
		expiration = &db.Expiration.Time
	}

	var password *string
	if db.Password.Valid {
		password = &db.Password.String
	}

	return paste.Paste{
		Reference:  db.Reference,
		Owner:      owner,
		Title:      db.Title,
		Content:    db.Content,
		Syntax:     syntax,
		Tags:       db.Tags,
		Creation:   db.CreatedAt.Time,
		Expiration: expiration,
		Public:     db.Public,
		Views:      int(db.Views.Int32),
		Password:   password,
	}
}

func (Paste) FromDomain(domain paste.Paste) *Paste {
	var owner pgtype.Text
	if domain.Owner != nil {
		owner = pgtype.Text{
			String: *domain.Owner,
			Valid:  true,
		}
	}

	var syntax pgtype.Text
	if domain.Syntax != "" {
		syntax = pgtype.Text{
			String: domain.Syntax,
			Valid:  true,
		}
	}

	var expiration pgtype.Timestamp
	if domain.Expiration != nil {
		expiration = pgtype.Timestamp{
			Time:  *domain.Expiration,
			Valid: true,
		}
	}

	var password pgtype.Text
	if domain.Password != nil {
		password = pgtype.Text{
			String: *domain.Password,
			Valid:  true,
		}
	}

	return &Paste{
		Reference:  domain.Reference,
		Owner:      owner,
		Title:      domain.Title,
		Content:    domain.Content,
		Syntax:     syntax,
		Tags:       domain.Tags,
		Expiration: expiration,
		Public:     domain.Public,
		Password:   password,
	}
}
