package authz

import (
	"encoding/json"
	"fmt"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// JSONRPCRequest models an incoming JSON-RPC 2.0 request payload from the MCP client.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ToolCallParams models the parameters of an MCP "tools/call" method.
type ToolCallParams struct {
	Name      string                     `json:"name"`
	Arguments map[string]json.RawMessage `json:"arguments,omitempty"`
}

// AdapterConfig configures the ext_authz to decision.Request adapter.
type AdapterConfig struct {
	DefaultWorkspaceID string
	DefaultBackendID   string
	IdentityMapper     *identity.Mapper
	ToolRegistry       *toolregistry.Registry
	ArgDeclarations    map[string]*argdecl.DeclarationSet
}

// AdapterError describes why an ext_authz check cannot be translated into a valid decision.Request.
type AdapterError struct {
	ReasonCode decision.ReasonCode
	Message    string
	Err        error
}

func (e *AdapterError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("authz adapter error (%s): %s: %v", e.ReasonCode, e.Message, e.Err)
	}
	return fmt.Sprintf("authz adapter error (%s): %s", e.ReasonCode, e.Message)
}

func (e *AdapterError) Unwrap() error {
	return e.Err
}
