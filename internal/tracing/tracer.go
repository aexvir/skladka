package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is a wrapper around the OpenTelemetry TracerProvider.
type Tracer struct {
	provider *sdk.TracerProvider
	tracer   trace.Tracer

	sampler   sdk.Sampler
	exporters []sdk.SpanExporter
}

// NewTracer creates a new tracer with the given service name, environment and version.
// Additional options can be provided to customize the tracer behavior.
//
// Returns the tracer instance, a shutdown function, and any error that occurred during initialization.
// The shutdown function should be called when the application is shutting down to ensure all spans are flushed.
func NewTracer(service, env, version string, opts ...TracerOption) (*Tracer, func(context.Context) error, error) {
	tracer := Tracer{
		sampler:   sdk.AlwaysSample(),
		exporters: make([]sdk.SpanExporter, 0),
	}

	for _, opt := range opts {
		if err := opt(&tracer); err != nil {
			return nil, nil, err
		}
	}

	// initialize provider with all configured exporters
	providerInitOptions := make([]sdk.TracerProviderOption, 0, len(tracer.exporters)+2)
	providerInitOptions = append(providerInitOptions,
		sdk.WithSampler(tracer.sampler),
		sdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(service),
				semconv.DeploymentEnvironmentName(env),
				semconv.ServiceVersion(version),
			),
		),
	)

	for _, exporter := range tracer.exporters {
		providerInitOptions = append(providerInitOptions, sdk.WithBatcher(exporter))
	}

	tracer.provider = sdk.NewTracerProvider(providerInitOptions...)
	tracer.tracer = tracer.provider.Tracer(service)

	return &tracer, func(ctx context.Context) error {
		if tracer.provider != nil {
			return tracer.provider.Shutdown(ctx)
		}
		return nil
	}, nil
}

// StartSpan starts a new span with the given operation name and attributes.
//
// The returned context contains the new span and should be passed to downstream functions.
// The returned function must be called when the operation is complete, typically using defer.
// The returned function accepts a pointer to an error which will be recorded if non-nil,
// along with any additional attributes to add to the span before it ends.
//
// Example usage:
//
//	ctx, finish := tracer.StartSpan(ctx, trace.SpanKindServer, "my-operation")
//	defer finish(&err)
func (t *Tracer) StartSpan(
	ctx context.Context,
	kind trace.SpanKind, operation string,
	attributes ...attribute.KeyValue,
) (context.Context, func(*error, ...attribute.KeyValue)) {
	ctx, span := t.tracer.Start(ctx, operation, trace.WithSpanKind(kind), trace.WithAttributes(attributes...))

	return ctx, func(err *error, extra ...attribute.KeyValue) {
		if len(extra) > 0 {
			span.SetAttributes(extra...)
		}
		// span status is unset by default
		// the opentelemetry semconv specify that unset should be treated as ok
		// so only error has to be set
		//
		// this is important because if other function got this span from context
		// and already set the status to error, but finish is being called with nil error
		// if we'd unconditionally set status to ok here it would override the error, as ok
		// has higher code
		if err != nil && *err != nil {
			span.SetStatus(codes.Error, (*err).Error())
			span.SetAttributes(
				semconv.ExceptionMessage((*err).Error()),
				semconv.ExceptionType(fmt.Sprintf("%T", *err)),
			)
			// todo: add stack trace from cause
		}
		span.End()
	}
}
