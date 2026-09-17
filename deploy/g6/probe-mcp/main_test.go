package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadStatusIncrementsOnce(t *testing.T) {
	backend := NewMCPServer()
	handler := backend.Handler()

	// 1. Send initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}`
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/", bytes.NewBufferString(initReq))
	r1.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected initialize 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Counter should still be 0
	if c := backend.GetCount(); c != 0 {
		t.Fatalf("expected count 0 after initialize, got %d", c)
	}

	// 2. Send valid tools/call for read_status
	callReq := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true}}}`
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/", bytes.NewBufferString(callReq))
	r2.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected tools/call 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Counter must now be exactly 1
	if c := backend.GetCount(); c != 1 {
		t.Fatalf("expected count exactly 1, got %d", c)
	}

	// Verify invocation record details
	invocations := backend.GetInvocations()
	if len(invocations) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(invocations))
	}
	if invocations[0].ToolName != "read_status" {
		t.Errorf("expected tool read_status, got %s", invocations[0].ToolName)
	}
}

func TestMalformedCallDoesNotIncrement(t *testing.T) {
	backend := NewMCPServer()
	handler := backend.Handler()

	// Send broken JSON
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/call",broken...`))
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	if backend.GetCount() != 0 {
		t.Errorf("malformed call must not increment count, got %d", backend.GetCount())
	}

	// Send unknown tool
	unknownReq := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"unknown_tool_exploit"}}`
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/", bytes.NewBufferString(unknownReq))
	r2.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w2, r2)

	if backend.GetCount() != 0 {
		t.Errorf("unknown tool call must not increment count, got %d", backend.GetCount())
	}

	var jsonResp map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &jsonResp); err != nil {
		t.Fatalf("failed to unmarshal unknown tool response: %v", err)
	}
	if jsonResp["error"] == nil {
		t.Errorf("expected error field in JSON-RPC response for unknown tool")
	}
}
