// Package metrics provides a thin wrapper around [OpenTelemetry]'s metrics sdk.
//
// The meter instance is meant to be initialized as a singleton and injected into the context
// during the initialization of the application. Both OTLP exporter via gRPC and Prometheus
// exporters are supported and can be enabled simultaneously.
//
// Every function that wants to record metrics should use the [metrics.FromContext] helper to obtain
// the meter instance in order to register its metrics. Metrics are defined using struct tags and must be registered before use.
//
// # example usage
//
//	// initialize meter instance
//	meter, err := metrics.NewMeter(
//		"service",
//		"environment",
//		"version",
//		metrics.WithOtlpExporter("localhost", 4317),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// inject meter into context
//	ctx := metrics.NewContext(context.Background(), meter)
//
// afterwards it can be used
//
//	// define metrics struct, usually at package level
//	type Metrics struct {
//		CreatePaste   metric.Int64Counter    `metric:"storage_paste_create_total,Number of pastes created"`
//		PasteSize     metric.Int64Histogram  `metric:"storage_paste_size_bytes,Size of pastes in bytes"`
//	}
//
//	// and reference them in the main package struct
//	type SqlStorage struct {
//		metrics *Metrics
//	}
//
//	// register metrics, usually as part of the constructor function
//	met := newStorageMetrics
//	if err := meter.Register(met); err != nil {
//		log.Fatal(err)
//	}
//
//	return &storage{
//		metrics: met
//	}
//
//	// then in any function the metrics can be used via the reference
//	s.metrics.CreatePaste.Add(ctx, 1)
//	s.metrics.PasteSize.Record(ctx, pasteSize)
//
// The metrics package automatically handles registration and management of OpenTelemetry metrics,
// making it easier to instrument code and collect metrics in production environments.
//
// [OpenTelemetry]: https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metrics
package metrics
