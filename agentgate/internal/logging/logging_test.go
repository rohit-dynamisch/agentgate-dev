package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewFiltersBelowConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelWarn, &buf)

	logger.Info("should not appear")
	logger.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Errorf("Info-level message was logged despite Warn minimum level: %q", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("Warn-level message was not logged: %q", out)
	}
}

func TestNewWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelInfo, &buf)

	logger.Info("starting agentgate", "environment", "test")

	var decoded map[string]any
	line := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(line), &decoded); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, line)
	}

	if decoded["msg"] != "starting agentgate" {
		t.Errorf("msg = %v, want %q", decoded["msg"], "starting agentgate")
	}
	if decoded["environment"] != "test" {
		t.Errorf("environment = %v, want %q", decoded["environment"], "test")
	}
}

func TestNewDefaultsToStdoutWhenWriterIsNil(t *testing.T) {
	logger := New(slog.LevelInfo, nil)
	if logger == nil {
		t.Fatal("New(..., nil) returned nil logger")
	}
}
