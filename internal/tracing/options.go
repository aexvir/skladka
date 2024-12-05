package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/logging"
)

// TracerOption allows customizing the tracer behaviour.
// Use option functions only as part of tracer initialization, do not
// call them outside of this process.
type TracerOption func(*Tracer) error

// WithSampleRate configures the sampling rate for traces.
// This option should only be used to initialize the tracer and it's not safe for
// concurrent use.
func WithSampleRate(value float64) TracerOption {
	return func(t *Tracer) error {
		t.sampler = sdktrace.TraceIDRatioBased(value)
		return nil
	}
}

// WithOtlpExporter configures the traces to be exported via OTLP to the specified endpoint.
// This option should only be used to initialize the tracer and it's not safe for
// concurrent use.
func WithOtlpExporter(ctx context.Context, hostname string, port int) TracerOption {
	return func(tracer *Tracer) (err error) {
		addr := fmt.Sprintf("%s:%d", hostname, port)
		logging.FromContext(ctx).Info("tracing.otlp", "initializing otlp trace exporter", "endpoint", addr)
		exporter, err := otlptrace.New(
			ctx,
			otlptracegrpc.NewClient(
				otlptracegrpc.WithEndpoint(addr),
				otlptracegrpc.WithInsecure(),
			),
		)
		if err != nil {
			return errors.Wrap(err, "failed to init otlp trace exporter")
		}

		tracer.exporters = append(tracer.exporters, exporter)
		return nil
	}
}

// WithStdOutExporter configures the tracer to print traces to standard output.
// This output is very verbose and logs a bunch of json without further processing.
// This option should only be used to initialize the tracer and it's not safe for
// concurrent use.
func WithStdOutExporter() TracerOption {
	return func(tracer *Tracer) error {
		out, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return errors.Wrap(err, "failed to init stdout trace exporter")
		}

		tracer.exporters = append(tracer.exporters, out)
		return nil
	}
}
