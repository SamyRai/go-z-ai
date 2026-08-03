// Command quota-usage inspects the GLM Coding Plan quota and recent usage
// for the configured API key. The same endpoints the official web dashboard
// and the 5+ community quota-monitor tools scrape — here exposed as a typed
// Go API for your own dashboards, alerts, and statusline integrations.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/quota-usage
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	ctx := context.Background()

	// Quota limit — the rolling windows that cap plan usage.
	limitResp, err := c.Quota().GetQuotaLimit(ctx)
	if err != nil {
		log.Fatalf("quota limit: %v", err)
	}
	fmt.Printf("=== Quota limits (plan: %s) ===\n", limitResp.Data.Level)
	for _, l := range limitResp.Data.Limits {
		fmt.Printf("  %-32s %s / %s  (%.0f%% used, resets %s)\n",
			l.WindowDescription(),
			formatCount(l.CurrentValue),
			formatCount(l.Usage),
			l.Percentage,
			time.UnixMilli(l.NextResetTime).Format("2006-01-02 15:04 MST"),
		)
	}

	// Usage over the current window — tokens consumed per model.
	since := time.Now().Add(-24 * time.Hour)
	usage, err := c.Quota().GetModelUsage(ctx, since, time.Now())
	if err != nil {
		log.Fatalf("model usage: %v", err)
	}
	fmt.Println("\n=== Model usage (last 24h) ===")
	fmt.Printf("  total: %d calls, %s tokens\n",
		usage.Data.TotalUsage.TotalModelCallCount,
		formatTokens(usage.Data.TotalUsage.TotalTokensUsage))
	for _, s := range usage.Data.ModelSummaryList {
		fmt.Printf("    %-20s %s tokens\n", s.ModelName, formatTokens(s.TotalTokens))
	}
}

func formatCount(n int) string {
	if n <= 0 {
		return "0"
	}
	return fmt.Sprintf("%d", n)
}

func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
