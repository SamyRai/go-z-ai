package usageview

import (
	"slices"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// UsageRow is one series of a usage heat map: a display label, the value per
// time bucket, and the window total.
type UsageRow struct {
	Label  string
	Values []int64
	Total  int64
}

// ModelRows returns per-model token rows in the server's sort order.
func ModelRows(d client.ModelUsageData) []UsageRow {
	series := slices.Clone(d.ModelDataList)
	slices.SortStableFunc(series, func(a, b client.ModelUsageSeries) int { return a.SortOrder - b.SortOrder })
	rows := make([]UsageRow, len(series))
	for i, s := range series {
		rows[i] = UsageRow{Label: s.ModelName, Values: s.TokensUsage, Total: s.TotalTokens}
	}
	return rows
}

// ToolRows returns per-tool call rows in the server's sort order, labeled in
// English: the series carry only the Chinese tool name, so the English one
// comes from the summary list (by tool code), falling back to the code.
func ToolRows(d client.ToolUsageData) []UsageRow {
	english := make(map[string]string, len(d.ToolSummaryList))
	for _, s := range d.ToolSummaryList {
		english[s.ToolCode] = s.ToolNameI18n
	}
	series := slices.Clone(d.ToolDataList)
	slices.SortStableFunc(series, func(a, b client.ToolUsageSeries) int { return a.SortOrder - b.SortOrder })
	rows := make([]UsageRow, len(series))
	for i, s := range series {
		label := english[s.ToolCode]
		if label == "" {
			label = s.ToolCode
		}
		rows[i] = UsageRow{Label: label, Values: s.UsageCount, Total: s.TotalUsageCount}
	}
	return rows
}

// RowTotal sums the rows' totals.
func RowTotal(rows []UsageRow) int64 {
	var total int64
	for _, r := range rows {
		total += r.Total
	}
	return total
}
