package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/example/sentinel-gateway/internal/health"
	"github.com/example/sentinel-gateway/internal/metrics"
	"github.com/example/sentinel-gateway/internal/middleware"
	"github.com/example/sentinel-gateway/internal/proxy"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", envOr("SENTINEL_CONFIG", "config/config.yaml"), "configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	observability := metrics.New()
	registry, err := proxy.NewRegistry(cfg, observability, logger)
	if err != nil {
		return fmt.Errorf("configure upstreams: %w", err)
	}
	ready := health.NewReadiness()

	api := registry.Handler()
	limiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.Routes, observability)
	api = middleware.Chain(api,
		middleware.Metrics(observability, cfg.Routes),
		middleware.Timeouts(cfg.Routes),
		registry.BreakerMiddleware(),
		middleware.JWTAuth(cfg.Auth),
		limiter.Middleware(),
		middleware.BodyLimit(cfg.Server.MaxBodyBytes),
	)

	mux := http.NewServeMux()
	mux.Handle(cfg.Routes["users"].Prefix, api)
	mux.Handle(cfg.Routes["payments"].Prefix, api)
	mux.Handle("/metrics", observability.Handler())
	mux.Handle("/healthz", health.LivenessHandler())
	mux.Handle("/readyz", health.ReadinessHandler(ready))
	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.StructuredLogger(logger),
		middleware.Recovery(logger),
	)

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           handler,
		ReadTimeout:       time.Duration(cfg.Server.ReadTimeout),
		ReadHeaderTimeout: time.Duration(cfg.Server.ReadHeaderTimeout),
		WriteTimeout:      time.Duration(cfg.Server.WriteTimeout),
		IdleTimeout:       time.Duration(cfg.Server.IdleTimeout),
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go limiter.Run(ctx)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("gateway started", "address", cfg.Server.Address)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
	case <-ctx.Done():
	}

	logger.Info("shutdown started")
	ready.Set(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout))
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
