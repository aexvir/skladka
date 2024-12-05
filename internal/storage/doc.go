// Package storage provides the persistence layer for the skladka service.
// It implements a PostgreSQL-based storage backend with support for:
//   - Storing and retrieving pastes with metadata
//   - Managing paste visibility (public/private)
//   - Handling paste expiration
//   - Reference generation and validation
//   - Content encryption for private pastes
//
// The package uses sqlc for type-safe SQL queries and includes metrics
// for monitoring database operations. It also integrates with the application's
// observability stack for logging and tracing.
//
// Example usage:
//
//	db, err := storage.NewPostgresStorage(ctx, cfg)
//	if err != nil {
//		return fmt.Errorf("failed to initialize storage: %w", err)
//	}
//
//	paste, err := db.GetPaste(ctx, reference)
//	if err != nil {
//		return fmt.Errorf("failed to retrieve paste: %w", err)
//	}
package storage
