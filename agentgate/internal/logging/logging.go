// Package logging builds AgentGate's structured logger.
//
// Per docs/TECH_STACK.md §2.7, AgentGate uses structured JSON logs via the
// standard library's log/slog — no external logging dependency is needed.
// Callers are responsible for not passing sensitive values (tool
// arguments, credentials) as log attributes; this package does not attempt
// to redact after the fact.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// New returns a JSON-structured logger writing to w at the given minimum
// level. If w is nil, it writes to os.Stdout.
func New(level slog.Level, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
