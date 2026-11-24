package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type claimsKey struct{}

type Claims struct {
	jwt.RegisteredClaims
}

func JWTAuth(settings config.AuthConfig) Middleware {
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(settings.Issuer),
		jwt.WithAudience(settings.Audience),
		jwt.WithExpirationRequired(),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			scheme, token, found := strings.Cut(header, " ")
			if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				writeUnauthorized(w)
				return
			}
			claims := &Claims{}
			parsed, err := jwt.ParseWithClaims(token, claims, func(parsed *jwt.Token) (interface{}, error) {
				if parsed.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(settings.Secret), nil
			}, parserOptions...)
			if err != nil || parsed == nil || !parsed.Valid || claims.Subject == "" {
				writeUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*Claims)
	return claims, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
