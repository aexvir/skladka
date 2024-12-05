package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

const (
	TagErrMessage string = "error.message"
	TagErrKind           = "error.kind"
	TagErrStack          = "error.stack"
)

type Logger struct {
	provider *sdk.LoggerProvider

	logger *slog.Logger
	level  *slog.LevelVar

	service string
	env     string
	version string

	processors []sdk.Processor
}

// NewLogger creates a new structured logger with the given service name, environment and version.
// It initializes the logger with OpenTelemetry integration and default settings. The logger can be
// customized using options like WithLevel, WithHandler, WithStdoutExporter, and WithOtlpExporter.
//
// Returns the logger instance, a shutdown function, and any error that occurred during initialization.
// The shutdown function should be called when the application is shutting down to ensure all logs are flushed.
func NewLogger(service, env, version string, opts ...LoggerOption) (*Logger, func(context.Context) error, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(service),
		semconv.DeploymentEnvironment(env),
		semconv.ServiceVersion(version),
	)

	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)

	logger := Logger{
		level:      lvl,
		service:    service,
		env:        env,
		version:    version,
		processors: make([]sdk.Processor, 0),
	}

	for _, opt := range opts {
		if err := opt(&logger); err != nil {
			return nil, nil, err
		}
	}

	providerInitOptions := make([]sdk.LoggerProviderOption, 0, len(logger.processors)+2)
	providerInitOptions = append(providerInitOptions, sdk.WithResource(res))

	for _, processor := range logger.processors {
		providerInitOptions = append(providerInitOptions, sdk.WithProcessor(processor))
	}

	logger.provider = sdk.NewLoggerProvider(providerInitOptions...)
	logger.logger = otelslog.NewLogger(service, otelslog.WithLoggerProvider(logger.provider))

	return &logger, func(ctx context.Context) error {
		if logger.provider != nil {
			return logger.provider.Shutdown(ctx)
		}
		return nil
	}, nil
}

// NewNopLogger returns a logger that discards all log messages. This is useful for testing
// or when logging is not needed. The returned logger implements all Logger methods but performs
// no operations.
func NewNopLogger() *Logger {
	return &Logger{
		logger: slog.New(
			slog.NewTextHandler(io.Discard, nil),
		),
	}
}

// With returns a new Logger with the given fields added to every log message.
// The fields are added as structured logging fields and will be present in all subsequent
// log entries made through the returned logger.
func (l *Logger) With(fields ...any) *Logger {
	clone := *l
	clone.logger = clone.logger.With(fields...)
	return &clone
}

// WithGroup creates a new Logger with the given group name. All subsequent log entries
// will be grouped under this name in the structured output.
func (l *Logger) WithGroup(name string) *Logger {
	clone := *l
	clone.logger = clone.logger.WithGroup(name)
	return &clone
}

// Info logs a message at INFO level with the given event type and optional fields.
// The event type is added as a structured field named "event" to help categorize
// and filter log entries.
func (l *Logger) Info(event, message string, fields ...any) {
	l.logger.Info(message, append(fields, slog.String("event", event))...)
}

// Debug logs a message at DEBUG level with the given event type and optional fields.
// Debug logs are only emitted if the logger level is set to DEBUG or lower.
// The event type is added as a structured field named "event".
func (l *Logger) Debug(event, message string, fields ...any) {
	l.logger.Debug(message, append(fields, slog.String("event", event))...)
}

// Warn logs a message at WARN level with the given event type and optional fields.
// The event type is added as a structured field named "event" to help categorize
// and filter log entries.
func (l *Logger) Warn(event, message string, fields ...any) {
	l.logger.Warn(message, append(fields, slog.String("event", event))...)
}

// Error logs a message at ERROR level with the given error, event type and optional fields.
// It automatically extracts and adds the following error details as structured fields:
//   - error.message: The error message from err.Error()
//   - error.kind: The type of the error
//   - error.stack: A formatted stack trace from the point of the error
func (l *Logger) Error(err error, event, message string, fields ...any) {
	stack := getStackTrace()

	l.logger.Error(
		message,
		append(
			fields,
			slog.String("event", event),
			slog.String(TagErrMessage, err.Error()),
			slog.String(TagErrKind, fmt.Sprintf("%T", err)),
			slog.String(TagErrStack, stack),
		)...,
	)
}

// getStackTrace returns a formatted stack trace starting from the caller of this function.
func getStackTrace() string {
	counters := make([]uintptr, 1024)
	// skip getStackTrace and Logger.Error functions from the stack
	n := runtime.Callers(4, counters)
	frames := runtime.CallersFrames(counters[:n])

	var buf bytes.Buffer

	for {
		frame, more := frames.Next()
		buf.WriteString(formatStackFrame(frame))
		if !more {
			break
		}
	}

	return buf.String()
}

// formatStackFrame formats a single stack frame into a human-readable string.
func formatStackFrame(frame runtime.Frame) string {
	return fmt.Sprintf(
		"%s\n\t%s:%d\n",
		funcname(frame.Function), frame.File, frame.Line,
	)
}

// funcname removes the path prefix component of a function's name reported by func.Name().
func funcname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}
