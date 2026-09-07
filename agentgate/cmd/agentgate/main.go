// Command agentgate is AgentGate's executable entry point.
//
// This scaffold (TASK-01-01) wires together typed configuration,
// structured logging, and the HTTP health/readiness surface, with clean
// startup and graceful shutdown. It does not implement authorization,
// identity resolution, policy evaluation, or audit — those are later
// tasks (see docs/PHASES/PHASE-01-MINIMAL-E2E-ENFORCEMENT.md).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dynamisch-LLC/agentgate/internal/config"
	"github.com/Dynamisch-LLC/agentgate/internal/httpserver"
	"github.com/Dynamisch-LLC/agentgate/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "agentgate: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logging.New(cfg.LogLevel, os.Stdout)
	logger.Info("starting agentgate",
		"environment", cfg.Environment,
		"http_addr", cfg.HTTPAddr,
	)

	srv := httpserver.New(cfg.HTTPAddr, logger)
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	logger.Info("http server listening", "addr", srv.Addr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("shutdown signal received, shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	logger.Info("agentgate stopped cleanly")
	return nil
}
