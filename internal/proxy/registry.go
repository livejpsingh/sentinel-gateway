package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/example/sentinel-gateway/internal/metrics"
	"github.com/example/sentinel-gateway/internal/middleware"
	"github.com/sony/gobreaker"
)

type Registry struct {
	cfg      config.Config
	metrics  *metrics.Metrics
	logger   *slog.Logger
	proxies  map[string]*httputil.ReverseProxy
	breakers map[string]*gobreaker.CircuitBreaker
}

func NewRegistry(cfg config.Config, collectors *metrics.Metrics, logger *slog.Logger) (*Registry, error) {
	registry := &Registry{
		cfg: cfg, metrics: collectors, logger: logger,
		proxies:  make(map[string]*httputil.ReverseProxy),
		breakers: make(map[string]*gobreaker.CircuitBreaker),
	}
	for _, name := range []string{"users", "payments"} {
		route := cfg.Routes[name]
		target, err := url.Parse(route.Upstream)
		if err != nil {
			return nil, fmt.Errorf("parse %s upstream: %w", name, err)
		}
		breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        name,
			MaxRequests: 1,
			Interval:    time.Minute,
			Timeout:     time.Duration(cfg.Circuit.OpenTimeout),
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= cfg.Circuit.FailureThreshold
			},
			OnStateChange: func(_ string, from, to gobreaker.State) {
				collectors.SetBreakerState(name, breakerStateValue(to))
				logger.Info("circuit breaker state changed",
					"upstream", name,
					"from", from.String(),
					"to", to.String(),
				)
			},
		})
		collectors.SetBreakerState(name, 0)
		proxy := httputil.NewSingleHostReverseProxy(target)
		defaultDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			defaultDirector(req)
			req.Header.Set("X-Request-ID", middleware.RequestIDFromContext(req.Context()))
			if traceparent := req.Header.Get("traceparent"); traceparent != "" {
				req.Header.Set("traceparent", traceparent)
			}
			if tracestate := req.Header.Get("tracestate"); tracestate != "" {
				req.Header.Set("tracestate", tracestate)
			}
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 1000
		transport.MaxIdleConnsPerHost = 256
		if route.MaxConnsPerHost < transport.MaxIdleConnsPerHost {
			transport.MaxIdleConnsPerHost = route.MaxConnsPerHost
		}
		transport.MaxConnsPerHost = route.MaxConnsPerHost
		transport.IdleConnTimeout = 90 * time.Second
		proxy.Transport = transport
		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			if errors.Is(req.Context().Err(), context.DeadlineExceeded) {
				middleware.WriteProxyError(w, http.StatusGatewayTimeout, "upstream timeout")
				return
			}
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				middleware.WriteProxyError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			logger.Error("upstream request failed",
				"upstream", name,
				"request_id", middleware.RequestIDFromContext(req.Context()),
				"error", err,
			)
			middleware.WriteProxyError(w, http.StatusBadGateway, "upstream unavailable")
		}
		registry.proxies[name] = proxy
		registry.breakers[name] = breaker
	}
	return registry, nil
}

func (r *Registry) Handler() http.Handler {
	mux := http.NewServeMux()
	usersPrefix := r.cfg.Routes["users"].Prefix
	paymentsPrefix := r.cfg.Routes["payments"].Prefix
	mux.Handle(usersPrefix, routeHandler(strings.TrimSuffix(usersPrefix, "/"), r.proxies["users"]))
	mux.Handle(paymentsPrefix, routeHandler(strings.TrimSuffix(paymentsPrefix, "/"), r.proxies["payments"]))
	return mux
}

func (r *Registry) BreakerMiddleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		usersHandler := middleware.CircuitBreaker("users", r.breakers["users"], r.metrics)(next)
		paymentsHandler := middleware.CircuitBreaker("payments", r.breakers["payments"], r.metrics)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, r.cfg.Routes["payments"].Prefix) {
				paymentsHandler.ServeHTTP(w, req)
				return
			}
			usersHandler.ServeHTTP(w, req)
		})
	}
}

func routeHandler(prefix string, proxy *httputil.ReverseProxy) http.Handler {
	return http.StripPrefix(prefix, proxy)
}

func breakerStateValue(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateHalfOpen:
		return 1
	case gobreaker.StateOpen:
		return 2
	default:
		return 0
	}
}
