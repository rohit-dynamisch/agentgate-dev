// Package httpserver provides AgentGate's HTTP surface.
//
// Today it only exposes liveness and readiness endpoints. Per
// docs/PROJECT_DEFINITION.md §2 (:8090 HTTP UI + API), later phases add the
// operator API/UI on the same listener — this package is the boundary
// where that will be wired in, not a placeholder for it.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
)

// Server serves AgentGate's HTTP endpoints.
type Server struct {
	addr   string
	logger *slog.Logger
	srv    *http.Server
	ln     net.Listener
	ready  atomic.Bool
}

// New constructs a Server bound to addr. It does not open a listener until
// Start is called.
func New(addr string, logger *slog.Logger) *Server {
	s := &Server{addr: addr, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	s.srv = &http.Server{Handler: mux}
	return s
}

// Start binds the listener and begins serving in the background. It
// returns as soon as the listener is bound, so a caller learns immediately
// whether the configured address is usable, rather than blocking for the
// server's entire lifetime.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("httpserver: listen on %s: %w", s.addr, err)
	}
	s.ln = ln
	s.ready.Store(true)

	go func() {
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("http server exited unexpectedly", "error", err)
		}
	}()

	return nil
}

// Addr returns the actual bound address (useful when the configured
// address uses an ephemeral port, e.g. in tests). It is empty until Start
// succeeds.
func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// Shutdown gracefully stops the server, waiting for in-flight requests to
// finish or for ctx to be done, whichever comes first.
func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	// Liveness: the process is up and able to handle a request at all.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	// Readiness: the server has bound its listener and has not begun
	// shutting down. This scaffold has no dependency (DB, policy store,
	// ...) to check yet — later tasks extend this as those dependencies
	// are introduced.
	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
