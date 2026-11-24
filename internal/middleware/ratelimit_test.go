package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/sentinel-gateway/internal/config"
	"github.com/example/sentinel-gateway/internal/metrics"
)

func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		wantStatus int
	}{
		{name: "first request is allowed", remoteAddr: "192.0.2.10:1234", wantStatus: http.StatusOK},
		{name: "same client is rejected after burst", remoteAddr: "192.0.2.10:1234", wantStatus: http.StatusTooManyRequests},
		{name: "different client has an independent bucket", remoteAddr: "192.0.2.11:1234", wantStatus: http.StatusOK},
	}
	limiter := NewRateLimiter(config.RateLimitConfig{
		RequestsPerSecond: 0.01,
		Burst:             1,
		MaxClients:        10,
		IdleTTL:           60,
		CleanupInterval:   60,
	}, map[string]config.RouteConfig{
		"users":    {Prefix: "/api/users/"},
		"payments": {Prefix: "/api/payments/"},
	}, metrics.New())
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
			request.RemoteAddr = test.remoteAddr
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantStatus == http.StatusTooManyRequests && response.Header().Get("Retry-After") == "" {
				t.Fatal("rejected response is missing Retry-After")
			}
		})
	}
}
