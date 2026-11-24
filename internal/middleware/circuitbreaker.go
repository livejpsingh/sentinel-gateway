package middleware

import (
	"errors"
	"net/http"

	"github.com/example/sentinel-gateway/internal/metrics"
	"github.com/sony/gobreaker"
)

func CircuitBreaker(upstream string, breaker *gobreaker.CircuitBreaker, collectors *metrics.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := breaker.Execute(func() (interface{}, error) {
				recorder := &responseRecorder{ResponseWriter: w}
				next.ServeHTTP(recorder, r)
				status := recorder.Status()
				if status >= http.StatusInternalServerError {
					if collectors != nil {
						collectors.UpstreamError(upstream)
					}
					return nil, errors.New("upstream returned server error")
				}
				return nil, nil
			})
			if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
				writeJSONError(w, http.StatusServiceUnavailable, "upstream circuit is open")
			}
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("{\"error\":\"" + message + "\"}\n"))
}
