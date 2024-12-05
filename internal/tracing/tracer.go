package tracing

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
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
				semconv.DeploymentEnvironment(env),
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
