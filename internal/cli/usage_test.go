package cli

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// captureStdout runs fn with os.Stdout redirected and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

// The quota table renderer (shared by `usage quota` and `accounts quota`) is
// the human-readable output most at risk from the output-format unification, so
// pin its structure: window description, usage line with count+percentage, and
// the per-tool breakdown.
func TestOutputQuotaLimitTable(t *testing.T) {
	quota := &client.QuotaLimitResponse{
		Success: true,
		Data: client.QuotaData{
			Level: "pro",
			Limits: []client.QuotaLimit{
				{
					Type:         string(client.QuotaTypeTimeLimit),
					Unit:         6,
					Number:       1,
					Usage:        100,
					CurrentValue: 40,
					Remaining:    60,
					Percentage:   40,
					UsageDetails: []client.ToolUsageDetail{
						{ModelCode: "web_search", Usage: 25},
					},
				},
			},
		},
	}

	out := captureStdout(t, func() {
		if err := outputQuotaLimit(quota, nil); err != nil {
			t.Fatalf("outputQuotaLimit: %v", err)
		}
	})

	for _, want := range []string{"PRO", "40/100", "40%", "60 remaining", "By tool", "web_search: 25"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected quota output to contain %q, got:\n%s", want, out)
		}
	}
}

// When a server timezone differing from the viewer's is supplied, the reset
// block gains a "Server:" line showing the same instant in the server's zone.
// When the zones match (or nil is passed), no Server line appears.
func TestOutputQuotaLimitServerLine(t *testing.T) {
	// Pin time.Local so the same-offset/match logic is deterministic.
	orig := time.Local
	time.Local = time.FixedZone("TEST+0", 0) // viewer at UTC+0
	defer func() { time.Local = orig }()

	reset := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	quota := &client.QuotaLimitResponse{
		Success: true,
		Data: client.QuotaData{
			Level: "pro",
			Limits: []client.QuotaLimit{{
				Type:          string(client.QuotaTypeTimeLimit),
				Unit:          6,
				Number:        1,
				Usage:         1000,
				CurrentValue:  100,
				Remaining:     900,
				Percentage:    10,
				NextResetTime: reset.UnixMilli(),
			}},
		},
	}

	cst := time.FixedZone("CST", 8*3600)
	out := captureStdout(t, func() {
		if err := outputQuotaLimit(quota, cst); err != nil {
			t.Fatalf("outputQuotaLimit: %v", err)
		}
	})
	if !strings.Contains(out, "Server:") {
		t.Errorf("differing serverTZ: expected a 'Server:' line, got:\n%s", out)
	}
	// The server line should show 17:00 CST (09:00 UTC + 8h).
	if !strings.Contains(out, "17:00:00 CST") {
		t.Errorf("differing serverTZ: expected server line at 17:00 CST, got:\n%s", out)
	}

	// Matching zones (viewer also at UTC+8) → no Server line.
	time.Local = time.FixedZone("VIEWER+8", 8*3600)
	out = captureStdout(t, func() {
		if err := outputQuotaLimit(quota, cst); err != nil {
			t.Fatalf("outputQuotaLimit: %v", err)
		}
	})
	if strings.Contains(out, "Server:") {
		t.Errorf("matching serverTZ: did not expect a 'Server:' line, got:\n%s", out)
	}

	// nil serverTZ → no Server line.
	time.Local = time.FixedZone("TEST+0", 0)
	out = captureStdout(t, func() {
		if err := outputQuotaLimit(quota, nil); err != nil {
			t.Fatalf("outputQuotaLimit: %v", err)
		}
	})
	if strings.Contains(out, "Server:") {
		t.Errorf("nil serverTZ: did not expect a 'Server:' line, got:\n%s", out)
	}
}
