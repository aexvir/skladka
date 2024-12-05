package logging

import "context"

const ctxKeyLogger = "logger"

// NewContext returns a new context.Context that carries the provided logger.
// This function should be used during application initialization to inject
// the logger instance into the context that will be passed throughout the
// application.
func NewContext(parent context.Context, logger *Logger) context.Context {
	return context.WithValue(parent, ctxKeyLogger, logger)
}

// BindContext creates a new context with a logger containing the provided fields.
// This is useful when you want to add context-specific fields to all subsequent
// log entries in a particular code path. The fields are added to a copy of the
// logger from the input context, and a new context carrying the updated logger
// is returned.
func BindContext(ctx context.Context, fields ...any) context.Context {
	logger := FromContext(ctx).With(fields...)
	return NewContext(ctx, logger)
}

// FromContext extracts the logger from the provided context.
// This is the recommended way to obtain a logger instance for logging.
// If no logger is found in the context, it returns a new no-op logger
// that safely discards all log messages.
func FromContext(ctx context.Context) *Logger {
	logger, ok := ctx.Value(ctxKeyLogger).(*Logger)

	if logger == nil || !ok {
		return NewNopLogger()
	}

	return logger
}
