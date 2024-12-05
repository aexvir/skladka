package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	. "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aexvir/skladka/internal/api"
	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/frontend"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/storage"
	"github.com/aexvir/skladka/internal/tracing"
)

func main() {
	rootctx, rootcancel := context.WithCancel(context.Background())

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrHelpWanted) {
			fmt.Println(err)
			return
		}
		panic(err)
	}

	logger, tracer, meter, otelshutdown, err := observability(rootctx, "skladka", cfg)
	if err != nil {
		panic(err)
	}

	rootctx = logging.NewContext(rootctx, logger)
	rootctx = tracing.NewContext(rootctx, tracer)
	rootctx = metrics.NewContext(rootctx, meter)

	db, err := storage.NewPostgresStorage(rootctx, cfg)
	if err != nil {
		logger.Error(err, "init.db", "failed to initialize database")
		return
	}

	router := NewRouter()
	router.Use(middleware.RequestID)
	router.Use(api.WithLogging(logger))
	router.Use(api.WithTracing(tracer))
	router.Use(middleware.Heartbeat("/health"))

	router.Mount("/", frontend.DashboardRouter(db))
	router.Handle("/metrics", metrics.Handler())

	server := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-exit
		logger.Info("cmd.serve", fmt.Sprintf("received signal: %v; initiating shutdown", sig))
		rootcancel()

		shutdownctx, shutdowncancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdowncancel()

		if err := server.Shutdown(shutdownctx); err != nil {
			logger.Error(err, "cmd.serve", "error during server shutdown")
		}

		if err := otelshutdown(shutdownctx); err != nil {
			logger.Error(err, "cmd.serve", "failed to shutdown observability components")
		}
	}()

	logger.Info("cmd.serve", "server listening on :3000")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err, "cmd.serve", "server error")
	}
}

// observability initializes logging, tracing and metrics for the application.
// returns the initialized components and a shutdown function that will cleanly shut down all components
// when called.
func observability(ctx context.Context, service string, cfg config.Config) (
	*logging.Logger, *tracing.Tracer, *metrics.Meter, func(context.Context) error, error,
) {
	loggeropts := make([]logging.LoggerOption, 0)
	if cfg.Logging.Enabled {
		loggeropts = append(loggeropts, logging.WithOtlpExporter(ctx, cfg.Logging.Host, cfg.Logging.Port))
	}

	meteropts := make([]metrics.MeterOption, 0)
	if cfg.Metrics.Enabled {
		meteropts = append(meteropts, metrics.WithOtlpExporter(ctx, cfg.Metrics.Host, cfg.Metrics.Port))
	}

	traceropts := make([]tracing.TracerOption, 0)
	if cfg.Tracing.Enabled {
		traceropts = append(traceropts, tracing.WithOtlpExporter(ctx, cfg.Tracing.Host, cfg.Tracing.Port))
	}

	if cfg.Environment == "dev" {
		// set up fancy logging for development environment
		loggeropts = append(
			loggeropts,
			logging.WithStdoutExporter(
				logging.NewFancyHandler(
					os.Stdout,
					&logging.FancyLoggerOptions{
						AddSource:  true,
						Level:      slog.LevelDebug,
						TimeFormat: time.Kitchen,
						ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
							// discard service, env and version tags locally
							// also proto
							if slices.Contains([]string{"service", "env", "version", "proto"}, attr.Key) {
								return slog.Attr{}
							}
							// otherwise don't touchy
							return attr
						},
					},
				),
			),
		)
	}

	logger, lgrshutdown, err := logging.NewLogger(service, cfg.Environment, config.BuildRevision, loggeropts...)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	meter, mtrshutdown, err := metrics.NewMeter(service, cfg.Environment, config.BuildRevision, meteropts...)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	tracer, trcshutdown, err := tracing.NewTracer(service, cfg.Environment, config.BuildRevision, traceropts...)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	return logger, tracer, meter,
		// Return a combined shutdown function that will shut down all components
		func(ctx context.Context) error {
			var errs []error

			if err := lgrshutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown logger: %w", err))
			}
			if err := mtrshutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown meter: %w", err))
			}
			if err := trcshutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown tracer: %w", err))
			}

			if len(errs) > 0 {
				return errors.Join(errs...)
			}

			return nil
		}, nil
}
