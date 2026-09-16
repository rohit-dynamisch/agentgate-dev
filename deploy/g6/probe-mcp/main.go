package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// InvocationRecord stores details of an accepted tool invocation.
type InvocationRecord struct {
	Timestamp  string            `json:"timestamp"`
	ToolName   string            `json:"tool_name"`
	Arguments  map[string]any    `json:"arguments"`
	Headers    map[string]string `json:"headers"`
	RawRequest string            `json:"raw_request"`
}

// MCPServer manages the mock MCP backend and atomic invocation counter.
type MCPServer struct {
	count       int64
	mu          sync.RWMutex
	invocations []InvocationRecord
}

func NewMCPServer() *MCPServer {
	return &MCPServer{
		invocations: make([]InvocationRecord, 0),
	}
}

func (s *MCPServer) GetCount() int64 {
	return atomic.LoadInt64(&s.count)
}

func (s *MCPServer) GetInvocations() []InvocationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := make([]InvocationRecord, len(s.invocations))
	copy(copied, s.invocations)
	return copied
}

func (s *MCPServer) Reset() {
	atomic.StoreInt64(&s.count, 0)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invocations = make([]InvocationRecord, 0)
}

// JSON-RPC models
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  any            `json:"result,omitempty"`
	Error   *jsonrpcError  `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Inspection endpoints
	mux.HandleFunc("/_g6/count", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"count":       s.GetCount(),
			"invocations": s.GetInvocations(),
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/_g6/reset", func(w http.ResponseWriter, r *http.Request) {
		s.Reset()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reset"))
	})

	// MCP JSON-RPC endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &jsonrpcError{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "initialize":
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"protocolVersion": "2026-07-28",
					"serverInfo": map[string]any{
						"name":    "g6-probe-mcp",
						"version": "1.0.0",
					},
					"capabilities": map[string]any{
						"tools": map[string]any{},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "tools/list":
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"tools": []map[string]any{
						{
							"name":        "read_status",
							"description": "Read operational status",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"verbose": map[string]any{"type": "boolean"},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "tools/call":
			var callParams struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &callParams); err != nil || callParams.Name != "read_status" {
				resp := jsonrpcResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &jsonrpcError{
						Code:    -32601,
						Message: fmt.Sprintf("tool '%s' not found or invalid params", callParams.Name),
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			// Valid tool call: increment counter and record invocation
			atomic.AddInt64(&s.count, 1)

			rec := InvocationRecord{
				Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
				ToolName:   callParams.Name,
				Arguments:  callParams.Arguments,
				Headers:    headers,
				RawRequest: string(bodyBytes),
			}
			s.mu.Lock()
			s.invocations = append(s.invocations, rec)
			s.mu.Unlock()

			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"content": []map[string]any{
						{
							"type": "text",
							"text": "OK: status operational",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jsonrpcError{
					Code:    -32601,
					Message: fmt.Sprintf("method '%s' not found", req.Method),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	})

	return mux
}

func main() {
	port := os.Getenv("G6_MCP_PORT")
	if port == "" {
		port = "9100"
	}
	countPort := os.Getenv("G6_COUNT_PORT")
	if countPort == "" {
		countPort = "9101"
	}

	server := NewMCPServer()
	handler := server.Handler()

	// If countPort differs from MCP port, we can run two servers or single mux on both
	if countPort != port {
		go func() {
			log.Printf("G6 MCP count/inspection server listening on :%s", countPort)
			if err := http.ListenAndServe(":"+countPort, handler); err != nil {
				log.Fatalf("count server failed: %v", err)
			}
		}()
	}

	log.Printf("G6 MCP backend server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
