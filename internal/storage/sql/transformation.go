package sql

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aexvir/skladka/internal/paste"
)

func (db Paste) ToDomain() paste.Paste {
	syntax := "plaintext"
	if db.Syntax.Valid {
		syntax = db.Syntax.String
	}

	var expiration *time.Time
	if db.Expiration.Valid {
		expiration = &db.Expiration.Time
	}

	return paste.Paste{
		Reference:  db.Reference,
		Title:      db.Title,
		Content:    db.Content,
		Syntax:     syntax,
		Tags:       db.Tags,
		Creation:   db.CreatedAt.Time,
		Expiration: expiration,
		Public:     db.Public,
		Views:      int(db.Views.Int32),
	}
}

func (Paste) FromDomain(domain paste.Paste) *Paste {
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

	return &Paste{
		Reference:  domain.Reference,
		Title:      domain.Title,
		Content:    domain.Content,
		Syntax:     syntax,
		Tags:       domain.Tags,
		Expiration: expiration,
		Public:     domain.Public,
	}
}
