package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

type requestIDKey struct{}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" || len(id) > 128 {
				var raw [16]byte
				if _, err := rand.Read(raw[:]); err != nil {
					http.Error(w, "request ID generation failed", http.StatusInternalServerError)
					return
				}
				raw[6] = (raw[6] & 0x0f) | 0x40
				raw[8] = (raw[8] & 0x3f) | 0x80
				id = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
					raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
			}
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
