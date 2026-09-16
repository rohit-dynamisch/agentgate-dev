package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Environment != defaultEnvironment {
		t.Errorf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.AuthzGRPCAddr != defaultAuthzGRPCAddr {
		t.Errorf("AuthzGRPCAddr = %q, want %q", cfg.AuthzGRPCAddr, defaultAuthzGRPCAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, defaultShutdownTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv(envPrefix+"ENV", "production")
	t.Setenv(envPrefix+"HTTP_ADDR", ":9999")
	t.Setenv(envPrefix+"AUTHZ_GRPC_ADDR", ":9005")
	t.Setenv(envPrefix+"LOG_LEVEL", "debug")
	t.Setenv(envPrefix+"SHUTDOWN_TIMEOUT", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Environment != "production" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "production")
	}
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9999")
	}
	if cfg.AuthzGRPCAddr != ":9005" {
		t.Errorf("AuthzGRPCAddr = %q, want %q", cfg.AuthzGRPCAddr, ":9005")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.ShutdownTimeout != 2*time.Second {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 2*time.Second)
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv(envPrefix+"LOG_LEVEL", "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for invalid log level")
	}
}

func TestLoadInvalidShutdownTimeout(t *testing.T) {
	t.Setenv(envPrefix+"SHUTDOWN_TIMEOUT", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for invalid shutdown timeout")
	}
}

func TestValidateRejectsEmptyHTTPAddr(t *testing.T) {
	cfg := Config{
		Environment:     defaultEnvironment,
		HTTPAddr:        "   ",
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: defaultShutdownTimeout,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error for empty HTTPAddr")
	}
}

func TestValidateRejectsNonPositiveShutdownTimeout(t *testing.T) {
	cfg := Config{
		Environment:     defaultEnvironment,
		HTTPAddr:        defaultHTTPAddr,
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: 0,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error for zero ShutdownTimeout")
	}
}
