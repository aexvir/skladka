// Package logging provides a thin wrapper around [OpenTelemetry]'s logging sdk with a [log/slog] interface.
//
// The logger instance is meant to be initialized as a singleton and injected into the context
// during the initialization of the application. It provides structured logging with support
// for different log levels, error handling with stack traces, and OpenTelemetry integration.
//
// Every function that wants to log messages should use the [logging.FromContext] helper to obtain
// the logger instance. The logger supports INFO, DEBUG, WARN and ERROR levels, with structured
// fields and automatic error details extraction.
//
// # example usage
//
//	// initialize logger instance
//	logger, err := logging.NewLogger(
//		"service",
//		"environment",
//		"version",
//		logging.WithLevel(slog.LevelDebug),
//	)
//	if err != nil {
//		return fmt.Errorf("failed to initialize logger: %w", err)
//	}
//
//	// inject logger into context
//	ctx := logging.NewContext(context.Background(), logger)
//
//	// anywhere else in the code
//	logger := logging.FromContext(ctx)
//	logger.Info("event_name", "Processing request", slog.String("request_id", "abc123"))
//	
//	// logging errors with automatic stack traces
//	if err := someOperation(); err != nil {
//		logger.Error(err, "operation_failed", "Operation failed", slog.String("op", "process"))
//	}
package logging
