// Command agentgate is AgentGate's executable entry point.
//
// Wires together typed configuration, structured logging, governance HTTP API,
// durable audit persistence, and the Envoy v3 ext_authz gRPC server for real MCP enforcement.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/audit"
	"github.com/Dynamisch-LLC/agentgate/internal/authz"
	"github.com/Dynamisch-LLC/agentgate/internal/config"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
	"github.com/Dynamisch-LLC/agentgate/internal/govapi"
	"github.com/Dynamisch-LLC/agentgate/internal/governanceintegration"
	"github.com/Dynamisch-LLC/agentgate/internal/httpserver"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/logging"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
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
		"authz_grpc_addr", cfg.AuthzGRPCAddr,
	)

	srv := httpserver.New(cfg.HTTPAddr, logger)

	var store policystore.Store
	var auditStore audit.Store
	if cfg.DatabaseURL != "" {
		pgStore, err := policystore.NewPostgresStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect postgres policy store: %w", err)
		}
		defer pgStore.Close()

		if err := pgStore.Migrate(context.Background()); err != nil {
			return fmt.Errorf("migrate postgres policy store: %w", err)
		}
		store = pgStore

		pgAuditStore, err := audit.NewPostgresStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect postgres audit store: %w", err)
		}
		defer pgAuditStore.Close()

		if err := pgAuditStore.Migrate(context.Background()); err != nil {
			return fmt.Errorf("migrate postgres audit store: %w", err)
		}
		auditStore = pgAuditStore

		srv.SetReadinessCheck(pgStore.Ping)
		logger.Info("connected to persistent postgres policy and audit store")
	} else {
		store = policystore.NewMemoryStore()
		defer store.Close()
		auditStore = audit.NewMemoryStore()
		defer auditStore.Close()
		logger.Info("using in-memory policy and audit store")
	}

	redactor := audit.NewRedactor(os.Getenv("AGENTGATE_AUDIT_SALT"), nil)
	auditedDecisionSvc := audit.NewAuditedDecisionService(nil, auditStore, redactor)

	policyMgr := policymanager.NewWithListener(store, auditedDecisionSvc)
	govIntegration := governanceintegration.NewGovernanceDecisionService(policyMgr)
	auditedDecisionSvc.SetEvaluator(govIntegration)

	// Seed default workspace baseline policy if none active
	ctxInit := context.Background()
	if _, err := policyMgr.GetActiveEngine("default"); err != nil {
		if _, err := policyMgr.CreateCandidateWithVersion(ctxInit, "default", "v1.0.0", fixturepolicy.CedarSource, "default baseline policy"); err == nil {
			_, _ = policyMgr.Activate(ctxInit, "default", "v1.0.0")
			logger.Info("seeded active baseline policy v1.0.0 for workspace 'default'")
		}
	}

	govHandler := govapi.NewHandler(policyMgr, cfg.AdminToken, govIntegration)
	govHandler.RegisterRoutes(srv.Mux())

	// Build ext_authz adapter components
	identityMapper, err := identity.NewMapper(identity.MapperConfig{
		AgentIDClaim:    "sub",
		RolesClaim:      "roles",
		OnBehalfOfClaim: "obo",
	})
	if err != nil {
		return fmt.Errorf("init identity mapper: %w", err)
	}

	readStatusSchema := []byte(`{"type":"object","properties":{"verbose":{"type":"boolean"}}}`)
	readStatusFP, _ := toolregistry.FingerprintSchema(readStatusSchema)

	writeStatusSchema := []byte(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}}}`)
	writeStatusFP, _ := toolregistry.FingerprintSchema(writeStatusSchema)

	toolReg, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{
		{
			ToolID:                toolregistry.ToolID{BackendID: "mcp-probe", ToolName: "read_status"},
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: readStatusFP,
		},
		{
			ToolID:                toolregistry.ToolID{BackendID: "default", ToolName: "read_status"},
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: readStatusFP,
		},
		{
			ToolID:                toolregistry.ToolID{BackendID: "mcp-probe", ToolName: "write_status"},
			Risk:                  toolregistry.RiskWrite,
			RegisteredFingerprint: writeStatusFP,
		},
		{
			ToolID:                toolregistry.ToolID{BackendID: "default", ToolName: "write_status"},
			Risk:                  toolregistry.RiskWrite,
			RegisteredFingerprint: writeStatusFP,
		},
		{
			ToolID:                toolregistry.ToolID{BackendID: "mcp-probe", ToolName: "admin_action"},
			Risk:                  toolregistry.RiskDestructive,
			RegisteredFingerprint: readStatusFP,
		},
		{
			ToolID:                toolregistry.ToolID{BackendID: "default", ToolName: "admin_action"},
			Risk:                  toolregistry.RiskDestructive,
			RegisteredFingerprint: readStatusFP,
		},
	})
	if err != nil {
		return fmt.Errorf("init tool registry: %w", err)
	}

	readStatusDecls, _ := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{
			Name:     "verbose",
			Type:     argdecl.ArgTypeBool,
			Required: false,
		},
	})
	writeStatusDecls, _ := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{
			Name:     "amount",
			Type:     argdecl.ArgTypeInt,
			Required: false,
		},
	})

	// In G6 single-workspace integration deployment, AllowStaticWorkspace permits fallback to
	// "default" when JWT does not declare a workspace_id claim. In multi-tenant environments,
	// setting this to false enforces fail-closed rejection for requests lacking trusted workspace claims.
	authzAdapter := authz.NewAdapter(authz.AdapterConfig{
		DefaultWorkspaceID:   "default",
		AllowStaticWorkspace: true,
		DefaultBackendID:     "mcp-probe",
		IdentityMapper:       identityMapper,
		ToolRegistry:         toolReg,
		ArgDeclarations: map[string]*argdecl.DeclarationSet{
			"read_status":  readStatusDecls,
			"write_status": writeStatusDecls,
		},
	})

	authzServer := authz.NewServer(authzAdapter, auditedDecisionSvc)

	// Start ext_authz gRPC Server
	grpcListener, err := net.Listen("tcp", cfg.AuthzGRPCAddr)
	if err != nil {
		return fmt.Errorf("listen authz grpc: %w", err)
	}
	defer grpcListener.Close()

	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, authzServer)

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("authz grpc server stopped with error", "err", err)
		}
	}()
	logger.Info("authz grpc server listening", "addr", cfg.AuthzGRPCAddr)

	// Start HTTP Server
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	logger.Info("http server listening", "addr", srv.Addr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("shutdown signal received, shutting down")

	grpcServer.GracefulStop()
	logger.Info("authz grpc server stopped cleanly")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	logger.Info("agentgate stopped cleanly")
	return nil
}
