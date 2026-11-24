package middleware

import (
	"net/http"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/example/sentinel-gateway/internal/metrics"
)

func Metrics(collectors *metrics.Metrics, routes map[string]config.RouteConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			collectors.ObserveRequest(r.Method, routeLabelForPath(r.URL.Path, routes), recorder.Status(), time.Since(start).Seconds())
		})
	}
}
