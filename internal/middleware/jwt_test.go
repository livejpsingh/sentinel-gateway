package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuth(t *testing.T) {
	settings := config.AuthConfig{
		Secret:   "a-test-secret-that-is-long-enough-for-hmac",
		Issuer:   "sentinel-test",
		Audience: "sentinel-api",
	}
	validToken := signTestToken(t, settings, jwt.RegisteredClaims{
		Subject:   "user-1",
		Issuer:    settings.Issuer,
		Audience:  []string{settings.Audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	wrongIssuer := signTestToken(t, settings, jwt.RegisteredClaims{
		Subject:   "user-1",
		Issuer:    "another-issuer",
		Audience:  []string{settings.Audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "valid HS256 token", token: validToken, wantStatus: http.StatusNoContent},
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "invalid signature", token: validToken + "x", wantStatus: http.StatusUnauthorized},
		{name: "wrong issuer", token: wrongIssuer, wantStatus: http.StatusUnauthorized},
	}
	handler := JWTAuth(settings)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ClaimsFromContext(r.Context()); !ok {
			t.Error("verified claims were not added to context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func signTestToken(t *testing.T, settings config.AuthConfig, claims jwt.RegisteredClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(settings.Secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return token
}
