package cli

import (
	"io"
	"os"
	"strings"
	"testing"

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
		if err := outputQuotaLimit(quota); err != nil {
			t.Fatalf("outputQuotaLimit: %v", err)
		}
	})

	for _, want := range []string{"PRO", "40/100", "40%", "60 remaining", "By tool", "web_search: 25"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected quota output to contain %q, got:\n%s", want, out)
		}
	}
}
