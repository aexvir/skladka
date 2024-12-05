package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/aexvir/skladka/internal/errors"
)

// MeterOption is a function that configures a Meter instance.
// Options are not safe for concurrent use during initialization.
// They should only be used when creating a new Meter instance.
type MeterOption func(*Meter) error

// WithOtlpExporter configures the metrics to be exported via OTLP to the specified endpoint.
// The exporter will connect to the specified hostname and port using gRPC without TLS.
// Metrics are exported every 5 seconds by default.
//
// This option is not safe for concurrent use during initialization. It should only be
// used when creating a new Meter instance via NewMeter.
func WithOtlpExporter(ctx context.Context, hostname string, port int) MeterOption {
	return func(m *Meter) error {
		addr := fmt.Sprintf("%s:%d", hostname, port)
		fmt.Println("initializing otlp metric exporter", "endpoint", addr)

		exp, err := otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(addr),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return errors.Wrap(err, "failed to create metrics otlp grpc exporter")
		}

		reader := sdkmetric.NewPeriodicReader(
			exp,
			sdkmetric.WithInterval(5*time.Second),
		)

		m.readers = append(m.readers, reader)
		return nil
	}
}

// WithPrometheusExporter configures the metrics to be exported in Prometheus format.
// This exporter is required to use the Handler function which exposes metrics via HTTP.
// The metrics will be available in Prometheus format at the /metrics endpoint.
//
// This option is not safe for concurrent use during initialization. It should only be
// used when creating a new Meter instance via NewMeter.
func WithPrometheusExporter() MeterOption {
	return func(m *Meter) error {
		exp, err := prometheus.New()
		if err != nil {
			return errors.Wrap(err, "creating Prometheus exporter")
		}

		m.readers = append(m.readers, exp)
		return nil
	}
}
