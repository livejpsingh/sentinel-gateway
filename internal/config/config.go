package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig           `yaml:"server"`
	Auth      AuthConfig             `yaml:"auth"`
	RateLimit RateLimitConfig        `yaml:"rate_limit"`
	Circuit   CircuitConfig          `yaml:"circuit_breaker"`
	Routes    map[string]RouteConfig `yaml:"routes"`
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

type ServerConfig struct {
	Address           string   `yaml:"address"`
	ReadTimeout       Duration `yaml:"read_timeout"`
	ReadHeaderTimeout Duration `yaml:"read_header_timeout"`
	WriteTimeout      Duration `yaml:"write_timeout"`
	IdleTimeout       Duration `yaml:"idle_timeout"`
	ShutdownTimeout   Duration `yaml:"shutdown_timeout"`
	MaxHeaderBytes    int      `yaml:"max_header_bytes"`
	MaxBodyBytes      int64    `yaml:"max_body_bytes"`
}

type AuthConfig struct {
	Secret   string `yaml:"secret"`
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64  `yaml:"requests_per_second"`
	Burst             int      `yaml:"burst"`
	MaxClients        int      `yaml:"max_clients"`
	IdleTTL           Duration `yaml:"idle_ttl"`
	CleanupInterval   Duration `yaml:"cleanup_interval"`
}

type CircuitConfig struct {
	FailureThreshold uint32   `yaml:"failure_threshold"`
	OpenTimeout      Duration `yaml:"open_timeout"`
}

type RouteConfig struct {
	Prefix          string   `yaml:"prefix"`
	Upstream        string   `yaml:"upstream"`
	Timeout         Duration `yaml:"timeout"`
	MaxConnsPerHost int      `yaml:"max_connections_per_host"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode YAML: %w", err)
	}
	if err := cfg.applyEnvironment(); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyEnvironment() error {
	stringOverrides := map[string]*string{
		"SENTINEL_ADDR":         &c.Server.Address,
		"SENTINEL_JWT_SECRET":   &c.Auth.Secret,
		"SENTINEL_JWT_ISSUER":   &c.Auth.Issuer,
		"SENTINEL_JWT_AUDIENCE": &c.Auth.Audience,
	}
	for name, target := range stringOverrides {
		if value, ok := os.LookupEnv(name); ok {
			*target = value
		}
	}

	durationOverrides := map[string]*Duration{
		"SENTINEL_READ_TIMEOUT":         &c.Server.ReadTimeout,
		"SENTINEL_READ_HEADER_TIMEOUT":  &c.Server.ReadHeaderTimeout,
		"SENTINEL_WRITE_TIMEOUT":        &c.Server.WriteTimeout,
		"SENTINEL_IDLE_TIMEOUT":         &c.Server.IdleTimeout,
		"SENTINEL_SHUTDOWN_TIMEOUT":     &c.Server.ShutdownTimeout,
		"SENTINEL_RATE_IDLE_TTL":        &c.RateLimit.IdleTTL,
		"SENTINEL_RATE_CLEANUP":         &c.RateLimit.CleanupInterval,
		"SENTINEL_BREAKER_OPEN_TIMEOUT": &c.Circuit.OpenTimeout,
	}
	for name, target := range durationOverrides {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			*target = Duration(parsed)
		}
	}

	for envName, routeName := range map[string]string{
		"SENTINEL_USERS_TIMEOUT":    "users",
		"SENTINEL_PAYMENTS_TIMEOUT": "payments",
	} {
		if value, ok := os.LookupEnv(envName); ok {
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("%s: %w", envName, err)
			}
			route := c.Routes[routeName]
			route.Timeout = Duration(parsed)
			c.Routes[routeName] = route
		}
	}
	for envName, routeName := range map[string]string{
		"SENTINEL_USERS_MAX_CONNECTIONS":    "users",
		"SENTINEL_PAYMENTS_MAX_CONNECTIONS": "payments",
	} {
		if value, ok := os.LookupEnv(envName); ok {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s: %w", envName, err)
			}
			route := c.Routes[routeName]
			route.MaxConnsPerHost = parsed
			c.Routes[routeName] = route
		}
	}
	for envName, routeName := range map[string]string{
		"SENTINEL_USERS_PREFIX":    "users",
		"SENTINEL_PAYMENTS_PREFIX": "payments",
	} {
		if value, ok := os.LookupEnv(envName); ok {
			route := c.Routes[routeName]
			route.Prefix = value
			c.Routes[routeName] = route
		}
	}
	for envName, routeName := range map[string]string{
		"SENTINEL_USERS_URL":    "users",
		"SENTINEL_PAYMENTS_URL": "payments",
	} {
		if value, ok := os.LookupEnv(envName); ok {
			route := c.Routes[routeName]
			route.Upstream = value
			c.Routes[routeName] = route
		}
	}

	intOverrides := map[string]*int{
		"SENTINEL_RATE_BURST":       &c.RateLimit.Burst,
		"SENTINEL_RATE_MAX_CLIENTS": &c.RateLimit.MaxClients,
		"SENTINEL_MAX_HEADER_BYTES": &c.Server.MaxHeaderBytes,
	}
	for name, target := range intOverrides {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			*target = parsed
		}
	}
	if value, ok := os.LookupEnv("SENTINEL_MAX_BODY_BYTES"); ok {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("SENTINEL_MAX_BODY_BYTES: %w", err)
		}
		c.Server.MaxBodyBytes = parsed
	}
	if value, ok := os.LookupEnv("SENTINEL_RATE_RPS"); ok {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("SENTINEL_RATE_RPS: %w", err)
		}
		c.RateLimit.RequestsPerSecond = parsed
	}
	if value, ok := os.LookupEnv("SENTINEL_BREAKER_FAILURE_THRESHOLD"); ok {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("SENTINEL_BREAKER_FAILURE_THRESHOLD: %w", err)
		}
		c.Circuit.FailureThreshold = uint32(parsed)
	}
	return nil
}

func (c Config) Validate() error {
	var problems []string
	if c.Server.Address == "" {
		problems = append(problems, "server.address is required")
	}
	if c.Server.ReadTimeout <= 0 || c.Server.ReadHeaderTimeout <= 0 ||
		c.Server.WriteTimeout <= 0 || c.Server.IdleTimeout <= 0 ||
		c.Server.ShutdownTimeout <= 0 {
		problems = append(problems, "all server timeouts must be positive")
	}
	if c.Server.MaxHeaderBytes <= 0 || c.Server.MaxBodyBytes <= 0 {
		problems = append(problems, "server header and body limits must be positive")
	}
	if len(c.Auth.Secret) < 32 {
		problems = append(problems, "auth.secret must contain at least 32 bytes")
	}
	if c.Auth.Issuer == "" || c.Auth.Audience == "" {
		problems = append(problems, "auth issuer and audience are required")
	}
	if c.RateLimit.RequestsPerSecond <= 0 || c.RateLimit.Burst <= 0 ||
		c.RateLimit.MaxClients <= 0 || c.RateLimit.IdleTTL <= 0 ||
		c.RateLimit.CleanupInterval <= 0 {
		problems = append(problems, "rate-limit settings must be positive")
	}
	if c.Circuit.FailureThreshold == 0 || c.Circuit.OpenTimeout <= 0 {
		problems = append(problems, "circuit-breaker settings must be positive")
	}
	for _, name := range []string{"users", "payments"} {
		route, exists := c.Routes[name]
		if !exists {
			problems = append(problems, "route "+name+" is required")
			continue
		}
		if route.Prefix == "" || !strings.HasPrefix(route.Prefix, "/") || !strings.HasSuffix(route.Prefix, "/") {
			problems = append(problems, "route "+name+" prefix must be an absolute path ending in slash")
		}
		upstream, err := url.ParseRequestURI(route.Upstream)
		if err != nil || upstream.Host == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") {
			problems = append(problems, "route "+name+" upstream must be an absolute http(s) URL")
		}
		if route.Timeout <= 0 {
			problems = append(problems, "route "+name+" timeout must be positive")
		}
		if route.MaxConnsPerHost <= 0 {
			problems = append(problems, "route "+name+" max_connections_per_host must be positive")
		}
	}
	usersPrefix, usersConfigured := c.Routes["users"]
	paymentsPrefix, paymentsConfigured := c.Routes["payments"]
	if usersConfigured && paymentsConfigured &&
		(strings.HasPrefix(usersPrefix.Prefix, paymentsPrefix.Prefix) ||
			strings.HasPrefix(paymentsPrefix.Prefix, usersPrefix.Prefix)) {
		problems = append(problems, "users and payments route prefixes must not overlap")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
