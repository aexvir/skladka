package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/aexvir/skladka/internal/attributes"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/tracing"
)

// WithMetrics returns a middleware that increments http core metrics on every request.
func WithMetrics(meter *metrics.Meter) func(http.Handler) http.Handler {
	met := new(
		struct {
			RequestCount    metrics.IntCounter     `metric:"http.requests.total,number of requests processed"`
			RequestDuration metrics.FloatHistogram `metric:"http.requests.duration,processing duration of requests,ms"`
		},
	)

	meter.Register(met)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				wrapper := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

				start := time.Now()
				defer func() {
					ctx := r.Context()

					attrs := []attribute.KeyValue{
						attributes.HttpReqMethod(r.Method),
						attributes.HttpRespStatusCode(wrapper.Status()),
						attributes.HttpRoute(
							chi.RouteContext(ctx).RoutePattern(),
						),
					}

					met.RequestCount.Add(ctx, 1, attrs...)
					met.RequestDuration.Record(ctx, float64(time.Since(start).Milliseconds()), attrs...)
				}()

				next.ServeHTTP(w, r)
			},
		)
	}
}

// WithTracing returns a middleware that creates a span for every request
// and propagates the tracing context down the line.
func WithTracing(tracer *tracing.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				wrapper := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

				operation := fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
				ctx, finish := tracer.StartSpan(r.Context(),
					trace.SpanKindServer, operation,
					attributes.HttpReqMethod(r.Method),
				)

				defer func() {
					finish(
						nil,
						attributes.HttpReqMethod(r.Method),
						attributes.HttpRespStatusCode(wrapper.Status()),
						attributes.HttpRoute(
							chi.RouteContext(r.Context()).RoutePattern(),
						),
					)
				}()

				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

// WithLogging returns a middleware that logs every request using the specified logger.
func WithLogging(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				wrapper := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
				// measure and log request
				start := time.Now()
				defer func() {
					// request uri and url path should be the same
					// but in cases like where the middleware modifies the path
					// that is no longer the case
					//
					// log the resolved path (a.k.a. after middleware modifications)
					// only in these cases
					var resolved slog.Attr
					if r.RequestURI != r.URL.Path {
						resolved = slog.String("http.route.resolved", r.URL.Path)
					}

					logger.Info(
						"api.serve",
						"request",
						attributes.HttpReqProtocol(r.Proto),
						attributes.HttpReqMethod(r.Method),
						attributes.HttpRoute(chi.RouteContext(r.Context()).RoutePattern()),
						resolved,
						attributes.HttpRespStatusCode(wrapper.Status()),
						attributes.HttpRespSizeBytes(wrapper.BytesWritten()),
						attributes.HttpRespDurationMilliseconds(int(time.Since(start).Milliseconds())),
					)
				}()

				next.ServeHTTP(wrapper, r)
			},
		)
	}
}

// WithPathPrefix prepends the path prefix to the request URL.
// This allows serving the same container on different subdomains without
// the need for a reverse proxy nor the user specifying the path.
// e.g. dash.example.com -> :3000/dash but api.example.com -> :3000/*
func WithPathPrefix(prefix string) func(http.Handler) http.Handler {
	if prefix == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				r.URL.Path = path.Join(prefix, r.URL.Path)
				next.ServeHTTP(w, r)
			},
		)
	}
}

// WithAllowedPaths only allows requests to paths starting with the given prefixes.
// Returns http404 if the path is not allowed.
// This allows serving the same container on different subdomains without every subdomain
// being able to serve every path.
// e.g. api.example.com/* -> :3000/* but api.example.com/dash -> http404
func WithAllowedPaths(whitelist []string) func(http.Handler) http.Handler {
	if len(whitelist) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				for _, prefix := range whitelist {
					if strings.HasPrefix(r.URL.Path, prefix) {
						next.ServeHTTP(w, r)
						return
					}
				}

				http.NotFound(w, r)
			},
		)
	}
}

// WithForbiddenPaths blocks requests to paths starting with the given prefixes.
// Returns http404 if the path is blacklisted.
// This allows serving the same container on different subdomains without every subdomain
// being able to serve every path.
func WithForbiddenPaths(blacklist []string) func(http.Handler) http.Handler {
	if len(blacklist) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				for _, prefix := range blacklist {
					if strings.HasPrefix(r.URL.Path, prefix) {
						http.NotFound(w, r)
						return
					}
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}
