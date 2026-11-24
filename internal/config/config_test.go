package config

import (
	"testing"
	"time"
)

func TestLoadParsesDurationsAndEnvironmentOverrides(t *testing.T) {
	t.Setenv("SENTINEL_ADDR", ":8181")
	t.Setenv("SENTINEL_PAYMENTS_TIMEOUT", "7s")
	t.Setenv("SENTINEL_PAYMENTS_MAX_CONNECTIONS", "128")

	cfg, err := Load("../../config/config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.Address != ":8181" {
		t.Fatalf("address = %q, want %q", cfg.Server.Address, ":8181")
	}
	if got, want := time.Duration(cfg.Routes["payments"].Timeout), 7*time.Second; got != want {
		t.Fatalf("payments timeout = %s, want %s", got, want)
	}
	if got, want := time.Duration(cfg.Server.ReadHeaderTimeout), 5*time.Second; got != want {
		t.Fatalf("read header timeout = %s, want %s", got, want)
	}
	if got, want := cfg.Routes["payments"].MaxConnsPerHost, 128; got != want {
		t.Fatalf("payments max connections = %d, want %d", got, want)
	}
}
