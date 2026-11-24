package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &responseRecorder{ResponseWriter: w}
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("request panic",
						"panic", recovered,
						"stack", string(debug.Stack()),
						"request_id", RequestIDFromContext(r.Context()),
					)
					if recorder.status == 0 {
						http.Error(recorder, "internal server error", http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}
