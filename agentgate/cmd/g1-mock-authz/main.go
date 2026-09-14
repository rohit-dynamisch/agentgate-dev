// Command g1-mock-authz is a temporary, non-production HTTP wrapper
// around AgentGate's real decision core (internal/decision), loaded with
// the canonical G1 fixture policy (internal/fixturepolicy). It exists
// solely so Gateway/MCP and QA/Security can integrate against the frozen
// G1 authorization contract before the production ext_authz gRPC service
// exists (docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md,
// AG-GO-G1-04).
//
// This binary is never started by cmd/agentgate and is not part of the
// production startup path — see
// docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md,
// Integration Rule 2 ("Mocks Are Temporary Parallelization Tools").
//
// Usage (see docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md for the
// full request/response contract and worked examples):
//
//	go run ./cmd/g1-mock-authz
//	curl -X POST localhost:8091/evaluate -d '{...}'
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
	"github.com/Dynamisch-LLC/agentgate/internal/logging"
	"github.com/Dynamisch-LLC/agentgate/internal/mockauthz"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "g1-mock-authz: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	addr := os.Getenv("G1_MOCK_AUTHZ_ADDR")
	if addr == "" {
		addr = ":8091"
	}

	engine, err := decision.NewEngine([]byte(fixturepolicy.CedarSource))
	if err != nil {
		return fmt.Errorf("load fixture policy: %w", err)
	}

	logger := logging.New(slog.LevelInfo, os.Stdout)
	logger.Info("g1-mock-authz starting", "addr", addr, "note", "non-production G1 contract mock — see docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md")

	handler := mockauthz.NewHandler(engine)
	return http.ListenAndServe(addr, handler)
}
