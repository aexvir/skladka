package main

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/aexvir/harness"
)

// generate migrations for changes in db schema
func Migration(ctx context.Context, name string) error {
	fmt.Printf("creating migration: %s\n", name)

	db, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("skladka"),
		postgres.WithUsername("popelar"),
		postgres.WithPassword("nope"),
		testcontainers.WithWaitStrategy(
			// First, we wait for the container to log readiness twice.
			// This is because it will restart itself after the first startup.
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			// Then, we wait for docker to actually serve the port on localhost.
			// For non-linux OSes like Mac and Windows, Docker or Rancher Desktop will have to start a separate proxy.
			// Without this, the tests will be flaky on those OSes!
			wait.ForListeningPort("5432/tcp"),
		),
	)

	if err != nil {
		return fmt.Errorf("failed to start database container: %w", err)
	}

	defer func() {
		if err := db.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate database container: %s\n", err)
		}
	}()

	connstr, err := db.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get db conn string: %w", err)
	}

	harness.Run(
		ctx,
		"atlas",
		harness.WithArgs(
			"migrate", "diff", name,
			"--dir", "file://internal/storage/sql/migrations",
			"--to", "file://internal/storage/sql/schema.sql",
			"--dev-url", connstr,
		),
	)

	return nil
}

// apply migration to the database specified as argument
func Migrate(ctx context.Context, connstr string) error {
	return harness.Run(
		ctx,
		"atlas",
		harness.WithArgs(
			"migrate", "apply",
			fmt.Sprintf("--url=%s", connstr),
			"--dir=file://internal/storage/sql/migrations",
		),
	)
}
