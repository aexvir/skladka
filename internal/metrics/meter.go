package metrics

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/aexvir/skladka/internal/errors"
)

// Meter is the central metrics registry that manages OpenTelemetry metrics.
// It provides a way to register and record metrics while abstracting away the
// underlying OpenTelemetry implementation details.
type Meter struct {
	provider *sdk.MeterProvider
	meter    metric.Meter

	resource *resource.Resource
	readers  []sdk.Reader
	mu       sync.RWMutex
}

// NewMeter creates a new metrics registry with the given service name, environment and version.
// Additional options can be provided to customize the metrics behavior, such as configuring
// exporters for OTLP or Prometheus. If no options are provided, a no-op meter will be returned.
//
// The service name, environment and version parameters are used to create a resource that
// identifies the service in the metrics backend.
//
// Returns the meter instance, a shutdown function, and any error that occurred during initialization.
// The shutdown function should be called when the application is shutting down to ensure all metrics are flushed.
func NewMeter(service, env, version string, opts ...MeterOption) (*Meter, func(context.Context) error, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(service),
		semconv.DeploymentEnvironmentName(env),
		semconv.ServiceVersion(version),
	)

	meter := &Meter{
		resource: res,
		readers:  make([]sdk.Reader, 0),
	}

	for _, opt := range opts {
		if err := opt(meter); err != nil {
			return nil, nil, errors.Wrap(err, "applying option")
		}
	}

	// initialize provider with all configured readers
	providerInitOptions := make([]sdk.Option, 0, len(meter.readers)+1)
	providerInitOptions = append(providerInitOptions, sdk.WithResource(meter.resource))

	for _, reader := range meter.readers {
		providerInitOptions = append(providerInitOptions, sdk.WithReader(reader))
	}

	meter.provider = sdk.NewMeterProvider(providerInitOptions...)
	meter.meter = meter.provider.Meter(service)

	return meter, func(ctx context.Context) error {
		if meter.provider != nil {
			return meter.provider.Shutdown(ctx)
		}
		return nil
	}, nil
}

// NewNoopMeter creates a new no-op metrics registry that silently discards all metrics.
// This is useful for testing or when metrics are not needed. All operations on the
// returned meter are no-ops, but the API remains the same.
func NewNoopMeter() *Meter {
	return &Meter{
		meter: noop.NewMeterProvider().Meter("noop"),
	}
}

// Register takes a struct with metric field tags and registers all metrics defined in it.
// The struct fields must be of OpenTelemetry metric types (Counter, Gauge, Histogram).
// The metric tag format is: `metric:"name,description[,unit]"`.
//
// Example struct with metric tags:
//
//	type Metrics struct {
//		Requests    metrics.IntCounter     `metric:"requests_total,Total number of requests"`
//		Duration    metrics.FloatHistogram `metric:"request_duration_seconds,Request duration,s"`
//	}
//
// If called on a no-op metrics instance, it will initialize all metrics as no-op metrics.
func (m *Meter) Register(spec any) error {
	val := reflect.ValueOf(spec)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("metrics struct must be a non-nil pointer")
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("metrics struct must be a struct")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	typ := elem.Type()
	for i := range elem.NumField() {
		field := elem.Field(i)
		if !field.CanSet() {
			continue
		}

		structField := typ.Field(i)
		tag := structField.Tag.Get("metric")
		if tag == "" {
			continue
		}

		if err := m.registerMetric(field, tag); err != nil {
			return errors.Wrapf(err, "registering metric %q", structField.Name)
		}
	}

	return nil
}

func (m *Meter) registerMetric(field reflect.Value, tag string) error {
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return errors.New("invalid metric tag format, expected 'name,description[,unit]'")
	}

	name, desc := parts[0], parts[1]
	unit := ""
	if len(parts) > 2 {
		unit = parts[2]
	}

	switch field.Type().String() {
	case "metrics.IntCounter":
		counter, err := m.meter.Int64Counter(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(IntCounter{counter}))
	case "metrics.IntHistogram":
		histogram, err := m.meter.Int64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(IntHistogram{histogram}))
	case "metrics.IntGauge":
		gauge, err := m.meter.Int64Gauge(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(IntGauge{gauge}))

	case "metrics.FloatCounter":
		counter, err := m.meter.Float64Counter(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(FloatCounter{counter}))
	case "metrics.FloatHistogram":
		histogram, err := m.meter.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(FloatHistogram{histogram}))
	case "metrics.FloatGauge":
		gauge, err := m.meter.Float64Gauge(name, metric.WithDescription(desc), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(FloatGauge{gauge}))

	default:
		return errors.Errorf("unsupported metric type: %s", field.Type())
	}

	return nil
}
