package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/example/sentinel-gateway/internal/metrics"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	settings config.RateLimitConfig
	routes   map[string]config.RouteConfig
	metrics  *metrics.Metrics
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	now      func() time.Time
}

func NewRateLimiter(settings config.RateLimitConfig, routes map[string]config.RouteConfig, collectors *metrics.Metrics) *RateLimiter {
	return &RateLimiter{
		settings: settings,
		routes:   routes,
		metrics:  collectors,
		clients:  make(map[string]*clientLimiter),
		now:      time.Now,
	}
}

func (l *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := l.now()
			key := clientIP(r)
			l.mu.Lock()
			entry, exists := l.clients[key]
			if !exists && len(l.clients) < l.settings.MaxClients {
				entry = &clientLimiter{limiter: rate.NewLimiter(rate.Limit(l.settings.RequestsPerSecond), l.settings.Burst)}
				l.clients[key] = entry
			}
			if entry != nil {
				entry.lastSeen = now
			}
			l.mu.Unlock()

			if entry == nil || !entry.limiter.AllowN(now, 1) {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("{\"error\":\"rate limit exceeded\"}\n"))
				if l.metrics != nil {
					l.metrics.RateLimitRejected(routeLabelForPath(r.URL.Path, l.routes))
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *RateLimiter) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(l.settings.CleanupInterval))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			l.removeIdle(now)
		}
	}
}

func (l *RateLimiter) removeIdle(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, entry := range l.clients {
		if now.Sub(entry.lastSeen) > time.Duration(l.settings.IdleTTL) {
			delete(l.clients, key)
		}
	}
}

func routeLabelForPath(path string, routes map[string]config.RouteConfig) string {
	for _, name := range []string{"users", "payments"} {
		prefix := routes[name].Prefix
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return "/api/" + name + "/*"
		}
	}
	return "unmatched"
}
