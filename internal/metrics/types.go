package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// IntCounter is an instrument that records increasing int values.
type IntCounter struct {
	metric.Int64Counter
}

// Add records a change to the counter.
//
// Use the WithAttributes option to include measurement attributes.
func (c IntCounter) Add(ctx context.Context, incr int, attributes ...attribute.KeyValue) {
	c.Int64Counter.Add(ctx, int64(incr), metric.WithAttributes(attributes...))
}

// IntHistogram is an instrument that records a distribution of int values.
type IntHistogram struct {
	metric.Int64Histogram
}

// Record adds an additional value to the distribution.
//
// Use the WithAttributes option to include measurement attributes.
func (h IntHistogram) Record(ctx context.Context, incr int, attributes ...attribute.KeyValue) {
	h.Int64Histogram.Record(ctx, int64(incr), metric.WithAttributes(attributes...))
}

// IntGauge is an instrument that records instantaneous int values.
type IntGauge struct {
	metric.Int64Gauge
}

// Record adds an additional value to the distribution.
//
// Use the WithAttributes option to include measurement attributes.
func (g IntGauge) Record(ctx context.Context, incr int, attributes ...attribute.KeyValue) {
	g.Int64Gauge.Record(ctx, int64(incr), metric.WithAttributes(attributes...))
}

// FloatCounter is an instrument that records increasing float64 values.
type FloatCounter struct {
	metric.Float64Counter
}

// Add records a change to the counter.
//
// Use the WithAttributes option to include measurement attributes.
func (c FloatCounter) Add(ctx context.Context, incr float64, attributes ...attribute.KeyValue) {
	c.Float64Counter.Add(ctx, incr, metric.WithAttributes(attributes...))
}

// FloatHistogram is an instrument that records a distribution of float64
// values.
type FloatHistogram struct {
	metric.Float64Histogram
}

// Record adds an additional value to the distribution.
//
// Use the WithAttributes option to include measurement attributes.
func (g FloatHistogram) Record(ctx context.Context, incr float64, attributes ...attribute.KeyValue) {
	g.Float64Histogram.Record(ctx, incr, metric.WithAttributes(attributes...))
}

// FloatGauge is an instrument that records instantaneous float64 values.
type FloatGauge struct {
	metric.Float64Gauge
}

// Record records the instantaneous value.
//
// Use the WithAttributes option to include measurement attributes.
func (g FloatGauge) Record(ctx context.Context, incr float64, attributes ...attribute.KeyValue) {
	g.Float64Gauge.Record(ctx, incr, metric.WithAttributes(attributes...))
}
