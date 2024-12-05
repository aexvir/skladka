package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an http.Handler that exposes metrics in Prometheus format.
// This handler should be mounted on the /metrics endpoint of your HTTP server
// to enable scraping by Prometheus. The handler is only available when the
// WithPrometheusExporter option is used during meter initialization.
func Handler() http.Handler {
	return promhttp.Handler()
}
