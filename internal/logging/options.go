package logging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdk "go.opentelemetry.io/otel/sdk/log"

	"github.com/aexvir/skladka/internal/errors"
)

type LoggerOption func(*Logger) error

// WithHandler configures the logger to use a custom slog.Handler.
// The handler will be wrapped with OpenTelemetry integration to ensure
// all logs are properly instrumented.
func WithHandler(handler slog.Handler) LoggerOption {
	return func(l *Logger) error {
		l.logger = otelslog.NewLogger(
			l.service,
			otelslog.WithLoggerProvider(l.provider),
		)
		return nil
	}
}

// WithLevel sets the minimum log level for the logger.
// Any log entry with a level below this threshold will be discarded.
// The default level is INFO. This option can be used to change the level
// at any time during the logger's lifecycle.
func WithLevel(level slog.Level) LoggerOption {
	return func(l *Logger) error {
		l.level.Set(level)
		return nil
	}
}

// WithStdoutExporter configures the logger to output logs to stdout
// using the provided handler for formatting. This is useful for local
// development and debugging.
//
// This option should only be used to initialize the tracer and it's not safe for
// concurrent use.
func WithStdoutExporter(handler slog.Handler) LoggerOption {
	return func(l *Logger) error {
		l.processors = append(l.processors, NewStdOutProcessor(handler))
		return nil
	}
}

// WithOtlpExporter configures the logger to export logs via OTLP/gRPC
// to the specified endpoint. This enables integration with OpenTelemetry
// collectors and observability platforms.
//
// This option should only be used to initialize the tracer and it's not safe for
// concurrent use.
func WithOtlpExporter(ctx context.Context, host string, port int) LoggerOption {
	return func(l *Logger) error {
		addr := fmt.Sprintf("%s:%d", host, port)
		exporter, err := otlploggrpc.New(
			ctx,
			otlploggrpc.WithEndpoint(addr),
			otlploggrpc.WithInsecure(),
		)
		if err != nil {
			return errors.Wrap(err, "failed to init otlp log exporter")
		}

		l.processors = append(l.processors, sdk.NewBatchProcessor(exporter, sdk.WithExportInterval(5*time.Second)))
		return nil
	}
}
