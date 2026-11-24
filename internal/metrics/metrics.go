package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry       *prometheus.Registry
	requests       *prometheus.CounterVec
	duration       *prometheus.HistogramVec
	limited        *prometheus.CounterVec
	breakerStates  *prometheus.GaugeVec
	upstreamErrors *prometheus.CounterVec
}

func New() *Metrics {
	m := &Metrics{registry: prometheus.NewRegistry()}
	m.requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sentinel_http_requests_total",
		Help: "Total HTTP requests served by route, method, and status.",
	}, []string{"method", "route", "status"})
	m.duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sentinel_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "route"})
	m.limited = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sentinel_rate_limit_rejected_total",
		Help: "Total requests rejected by the rate limiter.",
	}, []string{"route"})
	m.breakerStates = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sentinel_circuit_breaker_state",
		Help: "Circuit breaker state: 0=closed, 1=half-open, 2=open.",
	}, []string{"upstream"})
	m.upstreamErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sentinel_upstream_errors_total",
		Help: "Total failed upstream requests by upstream.",
	}, []string{"upstream"})
	m.registry.MustRegister(m.requests, m.duration, m.limited, m.breakerStates, m.upstreamErrors)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveRequest(method, route string, status int, seconds float64) {
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(seconds)
}

func (m *Metrics) RateLimitRejected(route string) {
	m.limited.WithLabelValues(route).Inc()
}

func (m *Metrics) SetBreakerState(upstream string, state float64) {
	m.breakerStates.WithLabelValues(upstream).Set(state)
}

func (m *Metrics) UpstreamError(upstream string) {
	m.upstreamErrors.WithLabelValues(upstream).Inc()
}
