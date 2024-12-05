package storage_test

import (
	"context"

	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/storage"
)

func ExampleNewPostgresStorage() error {
	ctx := context.Background()

	db, err := storage.NewPostgresStorage(
		ctx,
		config.Config{
			Postgres: config.Postgres{},
		},
	)

	if err != nil {
		return errors.Wrap(err, "failed to initialize storage")
	}

	_, err = db.GetPaste(ctx, "abc123")
	return nil
}
