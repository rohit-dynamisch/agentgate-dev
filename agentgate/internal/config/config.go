// Package config provides AgentGate's typed, validated startup
// configuration, read from the process environment.
//
// Only fields actually consumed by the current scaffold are present here.
// Later tasks (Cedar policy loading, Postgres audit/policy storage, the
// ext_authz gRPC listener, claims mapping, etc. — see
// docs/PHASES/PHASE-01-MINIMAL-E2E-ENFORCEMENT.md) will extend this struct
// as those packages are implemented. It is deliberately not pre-populated
// with fields nothing reads yet.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const envPrefix = "AGENTGATE_"

const (
	defaultEnvironment     = "development"
	defaultHTTPAddr        = ":8090"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 10 * time.Second
)

// Config holds AgentGate's startup configuration.
type Config struct {
	// Environment is a free-form deployment label (e.g. "development",
	// "staging", "production"). It is not validated against a fixed set;
	// it exists for logging/observability context only.
	Environment string

	// HTTPAddr is the listen address for AgentGate's HTTP surface.
	// Per docs/PROJECT_DEFINITION.md §2 this is the :8090 HTTP UI + API
	// port; today it only serves health/readiness.
	HTTPAddr string

	// LogLevel controls the minimum severity written by the structured
	// logger (see internal/logging).
	LogLevel slog.Level

	// ShutdownTimeout bounds how long graceful shutdown waits for
	// in-flight requests to finish before the process exits anyway.
	ShutdownTimeout time.Duration
}

// Load reads configuration from the process environment, applies defaults
// for anything unset, and validates the result. It returns a descriptive
// error rather than a partially-valid Config if validation fails, so
// AgentGate fails clearly at startup instead of running with bad config.
func Load() (Config, error) {
	var cfg Config

	cfg.Environment = getEnv(envPrefix+"ENV", defaultEnvironment)
	cfg.HTTPAddr = getEnv(envPrefix+"HTTP_ADDR", defaultHTTPAddr)

	level, err := parseLogLevel(getEnv(envPrefix+"LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	cfg.LogLevel = level

	timeoutStr := getEnv(envPrefix+"SHUTDOWN_TIMEOUT", defaultShutdownTimeout.String())
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return Config{}, fmt.Errorf("config: invalid %sSHUTDOWN_TIMEOUT %q: %w", envPrefix, timeoutStr, err)
	}
	cfg.ShutdownTimeout = timeout

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate fails clearly when required configuration is missing or
// nonsensical, rather than letting AgentGate start in an undefined state.
func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("config: %sHTTP_ADDR must not be empty", envPrefix)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("config: %sSHUTDOWN_TIMEOUT must be positive, got %s", envPrefix, c.ShutdownTimeout)
	}
	return nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid %sLOG_LEVEL %q (want debug|info|warn|error)", envPrefix, s)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
