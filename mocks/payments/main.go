package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	address := valueOr("PAYMENTS_ADDR", ":9002")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		delay, err := time.ParseDuration(os.Getenv("MOCK_DELAY"))
		if err == nil && delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-r.Context().Done():
				return
			case <-timer.C:
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "payments-service",
			"path":    r.URL.Path,
			"payment": map[string]interface{}{"id": "pay_demo_001", "status": "succeeded"},
		})
	})
	log.Printf("payments-service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func valueOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
