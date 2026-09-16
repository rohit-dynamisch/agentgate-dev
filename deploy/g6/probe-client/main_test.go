package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientInitializeAndCall(t *testing.T) {
	// Mock server mimicking an MCP gateway / backend
	var receivedCalls []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		receivedCalls = append(receivedCalls, method)

		w.Header().Set("Content-Type", "application/json")
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2026-07-28"}}`))
			return
		}
		if method == "tools/call" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"OK"}]}}`))
			return
		}
		http.Error(w, "unknown method", http.StatusBadRequest)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token")
	err := client.Initialize()
	if err != nil {
		t.Fatalf("expected Initialize to succeed, got: %v", err)
	}

	result, err := client.CallTool("read_status", map[string]any{"verbose": true})
	if err != nil {
		t.Fatalf("expected CallTool to succeed, got: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}

	if len(receivedCalls) != 2 || receivedCalls[0] != "initialize" || receivedCalls[1] != "tools/call" {
		t.Errorf("unexpected call sequence: %v", receivedCalls)
	}
}
