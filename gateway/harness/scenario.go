// Package harness is a verification harness, not a proxy.
//
// It exists because the real agentgateway binary cannot be executed in this environment
// (Docker Desktop's daemon is unreachable — see ../README.md "Docker / real-binary gap").
// It constructs exactly the JSON request shape documented in
// docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md (AG-GO-G1-04's "wire contract") and
// sends it over plain HTTP to a running instance of the real mock
// (agentgate/cmd/g1-mock-authz), so the request/response semantics can be proven end to end
// even though agentgateway itself cannot be run here.
//
// It deliberately:
//   - does no MCP transport, routing, or tool discovery (agentgateway's job, never rebuilt here);
//   - does not import any package from the agentgate Go module (this is its own Go module with
//     no dependency on agentgate/internal/decision or agentgate/internal/mockauthz) — it only
//     knows the documented wire JSON shape, exactly as an independent, out-of-process caller
//     (such as the real Rust agentgateway binary) would;
//   - never converts a DENY/error outcome into forwarding to a backend.
package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Expectation is the sanitized, human-authored expected outcome for one fixture.
type Expectation struct {
	HTTPStatus            int    `json:"http_status"`
	Decision              string `json:"decision"`
	Reason                string `json:"reason"`
	PolicyVersionNonEmpty bool   `json:"policy_version_nonempty"`
	BackendReached        bool   `json:"backend_reached"`
}

// Fixture is one G1 gateway-side scenario (see fixtures/README.md for the exact format).
// Exactly one of Request/RawBody is set.
type Fixture struct {
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Request     json.RawMessage `json:"request,omitempty"`
	RawBody     *string         `json:"raw_body,omitempty"`
	Expect      Expectation     `json:"expect"`

	// SourcePath is the file this fixture was loaded from (not part of the JSON shape).
	SourcePath string `json:"-"`
}

// Body returns the exact bytes that must be sent as the HTTP request body for this fixture.
func (f Fixture) Body() ([]byte, error) {
	if f.RawBody != nil {
		return []byte(*f.RawBody), nil
	}
	if len(f.Request) > 0 {
		return f.Request, nil
	}
	return nil, fmt.Errorf("fixture %q has neither request nor raw_body", f.Name)
}

// ResultWire mirrors the frozen mock response shape
// (agentgate/internal/mockauthz/wire.go's resultWire, documented in
// GO_BACKEND_G1_CONTRACT.md). It is redefined here independently, deliberately not imported
// from the agentgate module, so this harness proves the *wire contract* rather than merely
// reusing Go-internal types.
type ResultWire struct {
	Decision      string `json:"decision"`
	Reason        string `json:"reason"`
	Message       string `json:"message,omitempty"`
	PolicyVersion string `json:"policy_version,omitempty"`
	ExecutionID   string `json:"execution_id"`
}

// LoadFixtures reads every *.json file in dir, sorted by filename, into Fixtures.
func LoadFixtures(dir string) ([]Fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read fixtures dir %q: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	fixtures := make([]Fixture, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read fixture %q: %w", path, err)
		}
		var f Fixture
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("parse fixture %q: %w", path, err)
		}
		f.SourcePath = path
		fixtures = append(fixtures, f)
	}
	return fixtures, nil
}

// Outcome is what actually happened when a fixture was sent to the mock.
type Outcome struct {
	HTTPStatus int
	Result     ResultWire // zero value when HTTPStatus != 200 (transport-level rejection)
}

// Send POSTs a fixture's body to addr+"/evaluate" and reports what actually came back.
// It performs no interpretation of ALLOW/DENY beyond decoding the wire shape — the
// enforcement gate is WouldForwardToBackend, applied separately by the caller.
func Send(ctx context.Context, client *http.Client, addr string, f Fixture) (Outcome, error) {
	body, err := f.Body()
	if err != nil {
		return Outcome{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/evaluate", bytes.NewReader(body))
	if err != nil {
		return Outcome{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Outcome{}, fmt.Errorf("call mock at %s: %w", addr, err)
	}
	defer resp.Body.Close()

	out := Outcome{HTTPStatus: resp.StatusCode}
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out.Result); err != nil {
			return out, fmt.Errorf("decode mock response: %w", err)
		}
	}
	return out, nil
}

// WouldForwardToBackend is the enforcement gate this harness proves: continuation to a
// governed backend is permitted if, and only if, the mock returned HTTP 200 with
// decision == ALLOW. Every other outcome — DENY, any non-200 transport status, or a decode
// failure — must gate to false. There is no path in this function that can turn a
// non-ALLOW outcome into true.
func WouldForwardToBackend(o Outcome) bool {
	if o.HTTPStatus != http.StatusOK {
		return false
	}
	return o.Result.Decision == "ALLOW"
}

// DefaultMockAddr is the mock's documented default listen address
// (agentgate/cmd/g1-mock-authz, overridable via G1_MOCK_AUTHZ_ADDR).
const DefaultMockAddr = "http://localhost:8091"

// MockAddrFromEnv resolves the mock base URL the same way a human running these tickets
// would: G1_MOCK_AUTHZ_HARNESS_ADDR (this harness's own override) takes precedence, then
// DefaultMockAddr.
func MockAddrFromEnv() string {
	if v := os.Getenv("G1_MOCK_AUTHZ_HARNESS_ADDR"); v != "" {
		return v
	}
	return DefaultMockAddr
}

// NewHTTPClient returns a client with a short, explicit timeout — this harness must never
// hang indefinitely waiting on a mock that isn't there.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}
