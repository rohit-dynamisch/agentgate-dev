package harness

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestG1Scenarios is the AG-GW-G1-06 interoperability proof: every fixture in
// ../fixtures is sent, over real HTTP, to a REAL running instance of the frozen G1 mock
// (agentgate/cmd/g1-mock-authz — the real decision.Engine + real Cedar boundary +
// fixturepolicy.CedarSource, not a fake). Nothing here asserts an outcome without actually
// performing the HTTP round trip against that process.
//
// Reproduction (see ../README.md for the full walkthrough):
//
//	# terminal 1
//	cd agentgate && go run ./cmd/g1-mock-authz
//	# terminal 2
//	cd gateway/harness && go test ./... -v
//
// If the mock is not reachable, this test SKIPS (not silently passes) with instructions —
// it never reports a scenario as satisfied without a real response from the mock.
func TestG1Scenarios(t *testing.T) {
	addr := MockAddrFromEnv()
	client := NewHTTPClient()

	if err := ping(client, addr); err != nil {
		t.Skipf("g1-mock-authz not reachable at %s (%v).\n"+
			"Start it first: `cd agentgate && go run ./cmd/g1-mock-authz`\n"+
			"(override the address the mock listens on with G1_MOCK_AUTHZ_ADDR, and tell this "+
			"harness where to find it with G1_MOCK_AUTHZ_HARNESS_ADDR if it differs from %s).",
			addr, err, DefaultMockAddr)
	}

	fixtures, err := LoadFixtures("../fixtures")
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found in ../fixtures — nothing was verified")
	}

	seenCategories := map[string]bool{}

	for _, f := range fixtures {
		f := f
		t.Run(f.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			out, err := Send(ctx, client, addr, f)
			if err != nil {
				t.Fatalf("send fixture %s: %v", f.SourcePath, err)
			}

			if out.HTTPStatus != f.Expect.HTTPStatus {
				t.Errorf("http_status: got %d, want %d", out.HTTPStatus, f.Expect.HTTPStatus)
			}

			if f.Expect.HTTPStatus == http.StatusOK {
				if out.Result.Decision != f.Expect.Decision {
					t.Errorf("decision: got %q, want %q", out.Result.Decision, f.Expect.Decision)
				}
				if out.Result.Reason != f.Expect.Reason {
					t.Errorf("reason: got %q, want %q", out.Result.Reason, f.Expect.Reason)
				}
				gotNonEmpty := out.Result.PolicyVersion != ""
				if gotNonEmpty != f.Expect.PolicyVersionNonEmpty {
					t.Errorf("policy_version non-empty: got %v (value %q), want %v",
						gotNonEmpty, out.Result.PolicyVersion, f.Expect.PolicyVersionNonEmpty)
				}
				// execution_id must always be preserved unchanged, on every path
				// (GO_BACKEND_G1_CONTRACT.md AG-GO-G1-02: "preserved unchanged ... including
				// every early/structural denial"). Only checked when we sent a structured
				// request (raw_body fixtures may not even decode).
				if len(f.Request) > 0 {
					var reqEcho struct {
						ExecutionID string `json:"execution_id"`
					}
					_ = json.Unmarshal(f.Request, &reqEcho)
					if reqEcho.ExecutionID != "" && out.Result.ExecutionID != reqEcho.ExecutionID {
						t.Errorf("execution_id not preserved: sent %q, got back %q",
							reqEcho.ExecutionID, out.Result.ExecutionID)
					}
				}
			}

			// The enforcement gate this whole ticket exists to prove.
			gotForward := WouldForwardToBackend(out)
			if gotForward != f.Expect.BackendReached {
				t.Errorf("WouldForwardToBackend: got %v, want %v (decision=%q reason=%q http=%d) "+
					"-- a non-ALLOW outcome must NEVER gate a governed continuation",
					gotForward, f.Expect.BackendReached, out.Result.Decision, out.Result.Reason, out.HTTPStatus)
			}

			// Never allow an authorization FAILURE (evaluation_error, or any non-200 transport
			// rejection) to be mistaken for ALLOW -- the specific invariant AG-GW-G1-04 and the
			// project's fail-closed posture require.
			if f.Category == "evaluation_error" || f.Expect.HTTPStatus != http.StatusOK {
				if gotForward {
					t.Fatalf("SECURITY INVARIANT VIOLATION: category %q produced WouldForwardToBackend=true", f.Category)
				}
			}

			seenCategories[f.Category] = true
			t.Logf("category=%s http=%d decision=%q reason=%q policy_version=%q backend_reached=%v",
				f.Category, out.HTTPStatus, out.Result.Decision, out.Result.Reason, out.Result.PolicyVersion, gotForward)
		})
	}

	// AG-GW-G1-06 DoD: all mandatory G1 scenarios must actually have run.
	required := []string{"ALLOW", "DENY", "missing_identity", "unknown_tool", "malformed_input", "evaluation_error"}
	for _, c := range required {
		if !seenCategories[c] {
			t.Errorf("required G1 category %q was not exercised by any fixture", c)
		}
	}
}

// ping does a minimal reachability check without asserting anything about the mock's
// authorization behavior -- it only confirms the process is listening. An unrelated 404 for
// GET / (the mock only serves POST /evaluate) still proves the process is up.
func ping(client *http.Client, addr string) error {
	req, err := http.NewRequest(http.MethodGet, addr+"/", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
