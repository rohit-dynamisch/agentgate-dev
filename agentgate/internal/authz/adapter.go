package authz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/contextassembly"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

// Adapter translates an Envoy v3 ext_authz CheckRequest into a frozen decision.Request.
type Adapter struct {
	cfg AdapterConfig
}

// NewAdapter constructs a new Adapter instance.
func NewAdapter(cfg AdapterConfig) *Adapter {
	if cfg.DefaultWorkspaceID == "" {
		cfg.DefaultWorkspaceID = "default"
	}
	if cfg.DefaultBackendID == "" {
		cfg.DefaultBackendID = "default"
	}
	if cfg.ArgDeclarations == nil {
		cfg.ArgDeclarations = make(map[string]*argdecl.DeclarationSet)
	}
	return &Adapter{cfg: cfg}
}

// Adapt processes an ext_authz CheckRequest, validating caller identity,
// MCP JSON-RPC structure, tool governance classification, and declared arguments.
func (a *Adapter) Adapt(ctx context.Context, checkReq *authv3.CheckRequest) (decision.Request, *AdapterError) {
	if checkReq == nil || checkReq.Attributes == nil || checkReq.Attributes.Request == nil || checkReq.Attributes.Request.Http == nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "missing http request context in ext_authz check",
		}
	}

	httpReq := checkReq.Attributes.Request.Http
	headers := httpReq.Headers
	if headers == nil {
		headers = make(map[string]string)
	}

	// 1. Extract Body
	var bodyBytes []byte
	if len(httpReq.RawBody) > 0 {
		bodyBytes = httpReq.RawBody
	} else if len(httpReq.Body) > 0 {
		bodyBytes = []byte(httpReq.Body)
	}

	if len(bodyBytes) == 0 {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "missing request body in ext_authz check",
		}
	}

	// 2. Parse JSON-RPC MCP structure
	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(bodyBytes, &rpcReq); err != nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "malformed JSON-RPC body",
			Err:        err,
		}
	}

	if rpcReq.Method != "tools/call" {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    fmt.Sprintf("unsupported MCP method %q (only tools/call is authorized)", rpcReq.Method),
		}
	}

	if len(rpcReq.Params) == 0 {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "missing params in tools/call request",
		}
	}

	var toolParams ToolCallParams
	if err := json.Unmarshal(rpcReq.Params, &toolParams); err != nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "malformed tools/call params",
			Err:        err,
		}
	}

	toolName := strings.TrimSpace(toolParams.Name)
	if toolName == "" {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    "empty tool name in tools/call params",
		}
	}

	// 3. Extract Identity Claims (from JWT filter metadata or fallback headers)
	claims := a.extractClaims(checkReq, headers)
	if a.cfg.IdentityMapper == nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonInvalidIdentity,
			Message:    "identity mapper not configured",
		}
	}

	mappedIdentity, merr := a.cfg.IdentityMapper.Map(claims)
	if merr != nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonInvalidIdentity,
			Message:    merr.Error(),
			Err:        merr,
		}
	}

	// 4. Resolve WorkspaceID and ExecutionID
	workspaceID := getHeader(headers, "x-agentgate-workspace-id")
	if workspaceID == "" {
		if claimWs, ok := claims["workspace_id"]; ok && claimWs != "" {
			workspaceID = claimWs
		} else {
			workspaceID = a.cfg.DefaultWorkspaceID
		}
	}

	executionID := getHeader(headers, "x-execution-id")
	if executionID == "" {
		executionID = getHeader(headers, "x-request-id")
	}
	if executionID == "" {
		executionID = generateExecutionID()
	}

	// 5. Tool Registry Lookup & Governance
	backendID := getHeader(headers, "x-agentgate-backend-id")
	if backendID == "" && checkReq.Attributes.ContextExtensions != nil {
		backendID = checkReq.Attributes.ContextExtensions["backend_id"]
	}
	if backendID == "" {
		backendID = a.cfg.DefaultBackendID
	}

	toolID := toolregistry.ToolID{
		BackendID: backendID,
		ToolName:  toolName,
	}

	if a.cfg.ToolRegistry == nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonUnknownTool,
			Message:    "tool registry not configured",
		}
	}

	liveFP := toolregistry.SchemaFingerprint(getHeader(headers, "x-tool-fingerprint"))
	if liveFP == "" {
		liveFP = toolregistry.SchemaFingerprint(getHeader(headers, "x-agentgate-tool-fingerprint"))
	}
	if liveFP == "" && checkReq.Attributes.ContextExtensions != nil {
		liveFP = toolregistry.SchemaFingerprint(checkReq.Attributes.ContextExtensions["tool_fingerprint"])
	}

	govRecord := a.cfg.ToolRegistry.Lookup(toolID, liveFP)
	if !govRecord.Known {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonUnknownTool,
			Message:    fmt.Sprintf("tool %s is not registered in governance catalog", toolID),
		}
	}

	if govRecord.DriftStatus == toolregistry.DriftDetected {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonUnknownTool,
			Message:    fmt.Sprintf("tool %s schema drift detected", toolID),
		}
	}

	if err := govRecord.Validate(); err != nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonUnknownTool,
			Message:    err.Error(),
			Err:        err,
		}
	}

	// 6. Argument Resolution
	var resolvedArgs map[string]argdecl.ResolvedArg
	var declSet *argdecl.DeclarationSet
	if ds, ok := a.cfg.ArgDeclarations[toolID.String()]; ok {
		declSet = ds
	} else if ds, ok := a.cfg.ArgDeclarations[toolName]; ok {
		declSet = ds
	}

	if declSet != nil {
		rawArgs := make(map[string]argdecl.RawArgValue, len(toolParams.Arguments))
		for k, v := range toolParams.Arguments {
			rawArgs[k] = v
		}

		var rerr *argdecl.ResolutionError
		resolvedArgs, rerr = declSet.Resolve(rawArgs)
		if rerr != nil {
			return decision.Request{}, &AdapterError{
				ReasonCode: decision.ReasonMalformedRequest,
				Message:    rerr.Error(),
				Err:        rerr,
			}
		}
	}

	// 7. Context Assembly into frozen decision.Request
	assemblyIn := contextassembly.AssemblyInput{
		ExecutionID:      executionID,
		WorkspaceID:      workspaceID,
		MappedIdentity:   mappedIdentity,
		GovernanceRecord: govRecord,
		ResolvedArgs:     resolvedArgs,
	}

	decisionReq, assemErr := contextassembly.Assemble(assemblyIn)
	if assemErr != nil {
		return decision.Request{}, &AdapterError{
			ReasonCode: decision.ReasonMalformedRequest,
			Message:    assemErr.Error(),
			Err:        assemErr,
		}
	}

	return decisionReq, nil
}

func (a *Adapter) extractClaims(checkReq *authv3.CheckRequest, headers map[string]string) map[string]string {
	claims := make(map[string]string)

	// 1. Read claims from Envoy JWT filter metadata if present
	if checkReq.Attributes != nil && checkReq.Attributes.MetadataContext != nil && checkReq.Attributes.MetadataContext.FilterMetadata != nil {
		if jwtMetadata, ok := checkReq.Attributes.MetadataContext.FilterMetadata["envoy.filters.http.jwt_authn"]; ok && jwtMetadata != nil {
			for k, v := range jwtMetadata.Fields {
				claims[k] = structpbValueToString(v)
			}
		}
	}

	// 2. Overlay / fallback with verified identity headers
	if _, ok := claims["sub"]; !ok || claims["sub"] == "" {
		if val := getHeader(headers, "x-agent-id"); val != "" {
			claims["sub"] = val
		} else if val := getHeader(headers, "x-agentgate-agent-id"); val != "" {
			claims["sub"] = val
		}
	}

	if _, ok := claims["roles"]; !ok || claims["roles"] == "" {
		if val := getHeader(headers, "x-roles"); val != "" {
			claims["roles"] = val
		} else if val := getHeader(headers, "x-agentgate-roles"); val != "" {
			claims["roles"] = val
		}
	}

	if _, ok := claims["obo"]; !ok || claims["obo"] == "" {
		if val := getHeader(headers, "x-on-behalf-of"); val != "" {
			claims["obo"] = val
		} else if val := getHeader(headers, "x-obo"); val != "" {
			claims["obo"] = val
		} else if val := getHeader(headers, "x-agentgate-on-behalf-of"); val != "" {
			claims["obo"] = val
		}
	}

	return claims
}

func structpbValueToString(v *structpb.Value) string {
	if v == nil {
		return ""
	}
	switch k := v.Kind.(type) {
	case *structpb.Value_StringValue:
		return k.StringValue
	case *structpb.Value_NumberValue:
		return fmt.Sprintf("%v", k.NumberValue)
	case *structpb.Value_BoolValue:
		if k.BoolValue {
			return "true"
		}
		return "false"
	case *structpb.Value_ListValue:
		if k.ListValue == nil {
			return ""
		}
		var parts []string
		for _, item := range k.ListValue.Values {
			if s := structpbValueToString(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, identity.RolesSeparator)
	default:
		return ""
	}
}

func getHeader(headers map[string]string, key string) string {
	if v, ok := headers[key]; ok {
		return v
	}
	lowKey := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == lowKey {
			return v
		}
	}
	return ""
}

func generateExecutionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("exec-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}
