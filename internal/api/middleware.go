package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/tracing"
)

// WithTracing returns a middleware that adds tracing to all requests.
func WithTracing(tracer *tracing.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := tracing.NewContext(r.Context(), tracer)

				operation := fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
				ctx, finish := tracing.FromContext(ctx,
					trace.SpanKindServer,
					operation,
					semconv.HTTPMethod(r.Method),
					semconv.HTTPURL(r.URL.String()),
				)
				var err error
				defer finish(&err)

				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

// WithLogging enables logging service wide.
// To do that, it first injects the logger to the request context, so any
// downstream function can extract it as needed to log stuff.
// Additionally, it uses the logger to log every request received.
func WithLogging(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// inject logger to context
				ctx := logging.NewContext(r.Context(), logger)

				// measure and log request
				start := time.Now()
				wrapper := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
				defer func() {
					// request uri and url path should be the same
					// but in cases like where the middleware modifies the path
					// that is no longer the case
					//
					// log the resolved path (a.k.a. after middleware modifications)
					// only in these cases
					var resolved slog.Attr
					if r.RequestURI != r.URL.Path {
						resolved = slog.String("resolved", r.URL.Path)
					}

					logger.Info(
						"api.serve",
						"request",
						slog.String("proto", r.Proto),
						slog.String("method", r.Method),
						slog.String("url", r.RequestURI),
						resolved,
						slog.Int("status", wrapper.Status()),
						slog.Int("bytes", wrapper.BytesWritten()),
						slog.Int64("elapsed", time.Since(start).Milliseconds()),
					)
				}()

				next.ServeHTTP(wrapper, r.WithContext(ctx))
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
