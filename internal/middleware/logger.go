package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

func StructuredLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status(),
				"latency_ms", float64(time.Since(start).Microseconds())/1000,
				"request_id", RequestIDFromContext(r.Context()),
				"client_ip", clientIP(r),
			)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.Trim(r.RemoteAddr, "[]")
}
