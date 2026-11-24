package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	address := valueOr("USERS_ADDR", ":9001")
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
			"service": "users-service",
			"path":    r.URL.Path,
			"user":    map[string]interface{}{"id": 42, "name": "Ada Lovelace"},
		})
	})
	log.Printf("users-service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func valueOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
