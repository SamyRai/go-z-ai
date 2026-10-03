package usageview

import (
	"strings"
	"testing"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func TestSummarizeLimit(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, peakZone)
	credit := client.QuotaLimit{
		Type: client.QuotaTypeCreditLimit, Unit: client.UnitCodeHour, Number: 5,
		Usage: 2000, CurrentValue: 500, Remaining: 1500,
		NextResetTime: now.Add(2 * time.Hour).UnixMilli(),
	}
	s := SummarizeLimit(credit, BillingCredits, now, time.UTC)
	if s.Title != "5-hour credit window" || s.Used != 0.25 {
		t.Errorf("title/used = %q/%v", s.Title, s.Used)
	}
	if !strings.Contains(s.Counts, "500 / 2.0K used") || !strings.Contains(s.Reset, "(in 2h)") {
		t.Errorf("counts/reset = %q / %q", s.Counts, s.Reset)
	}
	if s.Pace == "" {
		t.Error("model-usage windows get a pace line")
	}

	idle := client.QuotaLimit{Type: client.QuotaTypeTokensLimit, Unit: client.UnitCodeHour, Number: 5, Percentage: 0}
	if s := SummarizeLimit(idle, BillingTokens, now, time.UTC); s.Counts != "" || !strings.Contains(s.Reset, "next request") {
		t.Errorf("idle percentage-only window = %+v", s)
	}

	mcp := client.QuotaLimit{Type: client.QuotaTypeTimeLimit, Unit: client.UnitCodeMonth, Number: 1,
		UsageDetails: []client.ToolUsageDetail{{ModelCode: "zread", Usage: 3}, {ModelCode: "web-reader", DisplayName: "Web Reader", Usage: 1}}}
	s = SummarizeLimit(mcp, BillingTokens, now, time.UTC)
	if s.Pace != "" || strings.Join(s.Tools, ";") != "zread: 3;Web Reader: 1" {
		t.Errorf("mcp lane = %+v", s)
	}
}

func TestToolRowsLabelsAndOrder(t *testing.T) {
	d := client.ToolUsageData{
		ToolDataList: []client.ToolUsageSeries{
			{ToolCode: "zread", ToolName: "读取", SortOrder: 2, TotalUsageCount: 1},
			{ToolCode: "search-prime", ToolName: "搜索", SortOrder: 1, TotalUsageCount: 4},
		},
		ToolSummaryList: []client.ToolUsageSummary{{ToolCode: "search-prime", ToolNameI18n: "Web Search"}},
	}
	rows := ToolRows(d)
	if rows[0].Label != "Web Search" || rows[1].Label != "zread" || RowTotal(rows) != 5 {
		t.Errorf("rows = %+v", rows)
	}
}
