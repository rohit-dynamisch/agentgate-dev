package httpserver_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/httpserver"
	"github.com/Dynamisch-LLC/agentgate/internal/logging"
)

func newTestLogger() *slog.Logger {
	return logging.New(slog.LevelError, nil)
}

func TestServerHealthAndReadyWhileServing(t *testing.T) {
	s := httpserver.New("127.0.0.1:0", newTestLogger())

	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	base := "http://" + s.Addr()

	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	resp2, err := http.Get(base + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("/readyz status = %d, want %d while serving", resp2.StatusCode, http.StatusOK)
	}
}

func TestServerUnreachableAfterShutdown(t *testing.T) {
	s := httpserver.New("127.0.0.1:0", newTestLogger())

	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	addr := s.Addr()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if _, err := http.Get("http://" + addr + "/healthz"); err == nil {
		t.Error("expected request after Shutdown to fail to connect, got nil error")
	}
}

func TestAddrEmptyBeforeStart(t *testing.T) {
	s := httpserver.New("127.0.0.1:0", newTestLogger())
	if got := s.Addr(); got != "" {
		t.Errorf("Addr() before Start() = %q, want empty string", got)
	}
}
