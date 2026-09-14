// Command g1report is a small CLI convenience wrapper around the harness package: it sends
// every fixture in gateway/fixtures to a running g1-mock-authz and prints a plain-text
// report table (request category, mock's decision/reason/policy_version, and whether the
// harness's enforcement gate would forward the call to a backend). It is a reporting tool
// for a human reading AG-GW-G1-06's evidence, not a test runner -- the automated assertions
// live in gateway/harness/g1_scenarios_test.go (`go test ./...`).
//
// Usage:
//
//	cd agentgate && go run ./cmd/g1-mock-authz &
//	cd gateway/harness && go run ./cmd/g1report
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	harness "github.com/Dynamisch-LLC/agentgate-gateway-harness"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "g1report: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	addr := harness.MockAddrFromEnv()
	client := harness.NewHTTPClient()

	fixtures, err := harness.LoadFixtures("../fixtures")
	if err != nil {
		return err
	}

	fmt.Printf("g1-gateway-harness -- reporting against mock at %s\n", addr)
	fmt.Printf("%-32s %-24s %-6s %-20s %-9s %-10s %s\n",
		"FIXTURE", "CATEGORY", "HTTP", "DECISION/REASON", "POLICYVER", "FORWARD?", "NOTE")

	failures := 0
	for _, f := range fixtures {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := harness.Send(ctx, client, addr, f)
		cancel()
		if err != nil {
			failures++
			fmt.Printf("%-32s %-24s ERROR: %v\n", f.Name, f.Category, err)
			continue
		}

		forward := harness.WouldForwardToBackend(out)
		note := ""
		if forward != f.Expect.BackendReached {
			failures++
			note = "MISMATCH vs expected fixture outcome"
		}
		polNonEmpty := "empty"
		if out.Result.PolicyVersion != "" {
			polNonEmpty = "present"
		}
		fmt.Printf("%-32s %-24s %-6d %-20s %-9s %-10v %s\n",
			f.Name, f.Category, out.HTTPStatus,
			out.Result.Decision+"/"+out.Result.Reason,
			polNonEmpty, forward, note)
	}

	if failures > 0 {
		return fmt.Errorf("%d fixture(s) did not match their expected outcome", failures)
	}
	fmt.Println("All fixtures matched their expected outcome.")
	return nil
}
