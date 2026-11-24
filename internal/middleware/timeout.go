package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
)

func Timeouts(routes map[string]config.RouteConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timeout := time.Duration(routes["users"].Timeout)
			if strings.HasPrefix(r.URL.Path, routes["payments"].Prefix) {
				timeout = time.Duration(routes["payments"].Timeout)
			}
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
