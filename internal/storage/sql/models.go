// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package sql

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Paste struct {
	ID         int64            `db:"id" json:"id"`
	Reference  string           `db:"reference" json:"reference"`
	Title      string           `db:"title" json:"title"`
	Content    string           `db:"content" json:"content"`
	Syntax     pgtype.Text      `db:"syntax" json:"syntax"`
	Tags       []string         `db:"tags" json:"tags"`
	Expiration pgtype.Timestamp `db:"expiration" json:"expiration"`
	Public     bool             `db:"public" json:"public"`
	CreatedAt  pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt  pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	DeletedAt  pgtype.Timestamp `db:"deleted_at" json:"deleted_at"`
	Views      pgtype.Int4      `db:"views" json:"views"`
}