// Package tracing provides a thin wrapper around [OpenTelemetry]'s tracing sdk.
//
// The tracer instance is meant to be initialized as a singleton and injected into the context
// during the initialization of the application. Only the otlp exporter via grpc is currently supported
// but extending this support should be fairly simple.
//
// Every function that wants to create a span should use the [tracing.FromContext] helper to obtain
// an instrumented context as well as its close function.
//
// # example usage
//
//	// initialize tracer instance
//	tracer, err := tracing.NewTracer("service", "environment", "version")
//		if err != nil {
//		return fmt.Errorf("failed to initialize tracing: %w", err)
//	}
//
//	// inject tracer into context
//	ctx := tracing.NewContext(context.Background(), tracer)
//
//	// anywhere else in the code that should be instrumented
//	var err error
//	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "domain.OperatioName")
//	defer finish(&err)
//
// [OpenTelemetry]: https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace
package tracing
