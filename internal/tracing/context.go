package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
)

const ctxKeyTracer = "tracer"

type CloseSpanFn func(err *error)

// NewContext returns a new context where the specified tracer has been
// injected to as value.
func NewContext(parent context.Context, tracer *Tracer) context.Context {
	return context.WithValue(parent, ctxKeyTracer, tracer)
}

// FromContext starts a new span and returns a new context instrumented with
// said span, as well as the function to close the span.
// The caller *must* call the close function.
// If no tracer instance is present in the context, this function is no-op.
func FromContext(
	ctx context.Context,
	kind trace.SpanKind, operation string,
	attributes ...attribute.KeyValue,
) (context.Context, func(err *error)) {
	tracer, ok := ctx.Value(ctxKeyTracer).(*Tracer)
	if tracer == nil || !ok {
		return ctx, func(err *error) {}
	}

	ctx, span := tracer.tracer.Start(ctx, operation, trace.WithSpanKind(kind), trace.WithAttributes(attributes...))

	return ctx, func(err *error) {
		span.SetStatus(codes.Ok, "")
		if err != nil && *err != nil {
			span.SetStatus(codes.Error, (*err).Error())
			span.SetAttributes(
				semconv.ExceptionMessage((*err).Error()),
			)
		}
		span.End()
	}
}
