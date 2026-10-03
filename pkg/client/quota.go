package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Quota limit types (QuotaLimit.Type). Credit-based plans (sold since
// 2026-07-30) meter the 5-hour and weekly windows in credits, with MCP tool
// calls billed from the same credits; legacy plans meter tokens and keep a
// separate monthly MCP-call lane. Treat Type as an open set.
const (
	QuotaTypeCreditLimit = "CREDIT_LIMIT" // credits (credit-based plans)
	QuotaTypeTokensLimit = "TOKENS_LIMIT" // tokens, percentage only (legacy plans)
	QuotaTypeTimeLimit   = "TIME_LIMIT"   // MCP tool calls (legacy plans)
)

// Window unit codes (QuotaLimit.Unit).
const (
	UnitCodeHour  = 3
	UnitCodeMonth = 5
	UnitCodeWeek  = 6
)

// QuotaService reads GLM Coding Plan quota and usage from the monitor API.
// The endpoints are not in the public API reference but are the ones Z.AI's
// own usage plugin calls.
type QuotaService struct {
	client *Client
}

// QuotaLimitResponse is the GET /usage/quota/limit response.
type QuotaLimitResponse = Envelope[QuotaData]

// QuotaData is the plan tier and its quota windows.
type QuotaData struct {
	Limits []QuotaLimit `json:"limits"`
	Level  string       `json:"level"` // lite, pro, max
}

// Exhausted reports whether any model-usage window has hit its limit, i.e.
// the plan cannot serve requests until that window resets.
func (d QuotaData) Exhausted() bool {
	for _, l := range d.Limits {
		if l.IsModelLimit() && l.UsedFraction() >= 1 {
			return true
		}
	}
	return false
}

// QuotaLimit is one quota window. On credit plans Usage is the allowance,
// CurrentValue the amount used, and Remaining what is left; legacy token
// windows report Percentage only.
type QuotaLimit struct {
	Type          string            `json:"type"`                   // QuotaType* constant
	Unit          int               `json:"unit"`                   // UnitCode* constant
	Number        int               `json:"number"`                 // units per window (5 for the 5-hour window)
	Usage         int               `json:"usage"`                  // allowance; 0 when not reported
	CurrentValue  int               `json:"currentValue"`           // used so far
	Remaining     int               `json:"remaining"`              // allowance left
	Percentage    float64           `json:"percentage"`             // server-rounded 0-100 (may overshoot)
	NextResetTime int64             `json:"nextResetTime"`          // epoch ms; 0 while an idle window hasn't started
	UsageDetails  []ToolUsageDetail `json:"usageDetails,omitempty"` // per-tool breakdown (TIME_LIMIT)
}

// ToolUsageDetail represents tool usage breakdown for TIME_LIMIT quotas
// ModelCode identifies which tool: "search-prime", "web-reader", "zread"
type ToolUsageDetail struct {
	ModelCode   string `json:"modelCode"`
	DisplayName string `json:"displayName,omitempty"`
	Usage       int    `json:"usage"`
}

// IsModelLimit reports whether the window meters model usage (credits or
// tokens), as opposed to the legacy MCP-call lane.
func (q *QuotaLimit) IsModelLimit() bool {
	return q.Type == QuotaTypeCreditLimit || q.Type == QuotaTypeTokensLimit
}

// UsedFraction is the share of the window consumed (0..1+), from the exact
// counts when the server reports an allowance, else from Percentage.
func (q *QuotaLimit) UsedFraction() float64 {
	if q.Usage > 0 {
		return float64(q.CurrentValue) / float64(q.Usage)
	}
	return q.Percentage / 100
}

// ResetTime returns the next reset time, or the zero time when unknown.
func (q *QuotaLimit) ResetTime() time.Time {
	if q.NextResetTime == 0 {
		return time.Time{}
	}
	return time.UnixMilli(q.NextResetTime)
}

// WindowDuration returns the window length: Number hours for the hourly
// unit, and one week / one month for those units regardless of Number (the
// server has been seen reporting a week as both (6,1) and (6,7)). Zero for
// an unknown unit.
func (q *QuotaLimit) WindowDuration() time.Duration {
	switch q.Unit {
	case UnitCodeHour:
		return time.Duration(max(q.Number, 1)) * time.Hour
	case UnitCodeWeek:
		return 7 * 24 * time.Hour
	case UnitCodeMonth:
		return 30 * 24 * time.Hour
	}
	return 0
}

// WindowStart returns when the current window began (reset time minus window
// duration), or the zero time when either is unknown.
func (q *QuotaLimit) WindowStart() time.Time {
	reset, d := q.ResetTime(), q.WindowDuration()
	if reset.IsZero() || d == 0 {
		return time.Time{}
	}
	return reset.Add(-d)
}

// WindowDescription is a human-readable name for the window, such as
// "5-hour credit window" or "monthly MCP tool quota".
func (q *QuotaLimit) WindowDescription() string {
	var span string
	switch q.Unit {
	case UnitCodeHour:
		span = fmt.Sprintf("%d-hour", max(q.Number, 1))
	case UnitCodeWeek:
		span = "weekly"
	case UnitCodeMonth:
		span = "monthly"
	default:
		span = fmt.Sprintf("unit-%d×%d", q.Unit, q.Number)
	}
	switch q.Type {
	case QuotaTypeCreditLimit:
		return span + " credit window"
	case QuotaTypeTokensLimit:
		return span + " token window"
	case QuotaTypeTimeLimit:
		return span + " MCP tool quota"
	}
	return fmt.Sprintf("%s %s window", span, q.Type)
}

// ModelUsageResponse represents model usage statistics for a time window,
// bucketed daily or hourly depending on the requested range (see
// ModelUsageData.Granularity). Verified against the live API — the response
// is a time-series object, not a flat list.
type ModelUsageResponse = Envelope[ModelUsageData]

// ModelUsageData holds parallel time-series arrays (one entry per XTime
// bucket) plus a per-model breakdown and totals for the requested window.
type ModelUsageData struct {
	XTime            []string            `json:"x_time"`
	ModelCallCount   []int64             `json:"modelCallCount"`
	TokensUsage      []int64             `json:"tokensUsage"`
	TotalUsage       ModelUsageTotal     `json:"totalUsage"`
	ModelDataList    []ModelUsageSeries  `json:"modelDataList"`
	ModelSummaryList []ModelUsageSummary `json:"modelSummaryList"`
	Granularity      string              `json:"granularity"` // "daily" or "hourly"
}

// ModelUsageTotal is the window-wide aggregate.
type ModelUsageTotal struct {
	TotalModelCallCount int64               `json:"totalModelCallCount"`
	TotalTokensUsage    int64               `json:"totalTokensUsage"`
	ModelSummaryList    []ModelUsageSummary `json:"modelSummaryList"`
}

// ModelUsageSummary is one model's window-wide total.
type ModelUsageSummary struct {
	ModelName   string `json:"modelName"`
	TotalTokens int64  `json:"totalTokens"`
	SortOrder   int    `json:"sortOrder"`
}

// ModelUsageSeries is one model's per-bucket token usage across XTime.
type ModelUsageSeries struct {
	ModelName   string  `json:"modelName"`
	SortOrder   int     `json:"sortOrder"`
	TokensUsage []int64 `json:"tokensUsage"`
	TotalTokens int64   `json:"totalTokens"`
}

// ToolUsageResponse represents MCP tool (web search/reader/zread) usage
// statistics for a time window, same bucketed time-series shape as
// ModelUsageResponse.
type ToolUsageResponse = Envelope[ToolUsageData]

// ToolUsageData holds parallel time-series arrays plus a per-tool breakdown
// and totals for the requested window.
type ToolUsageData struct {
	XTime              []string           `json:"x_time"`
	NetworkSearchCount []int64            `json:"networkSearchCount"`
	WebReadMcpCount    []int64            `json:"webReadMcpCount"`
	ZreadMcpCount      []int64            `json:"zreadMcpCount"`
	TotalUsage         ToolUsageTotal     `json:"totalUsage"`
	ToolDataList       []ToolUsageSeries  `json:"toolDataList"`
	ToolSummaryList    []ToolUsageSummary `json:"toolSummaryList"`
	Granularity        string             `json:"granularity"` // "daily" or "hourly"
}

// ToolUsageTotal is the window-wide aggregate.
type ToolUsageTotal struct {
	TotalNetworkSearchCount int64              `json:"totalNetworkSearchCount"`
	TotalWebReadMcpCount    int64              `json:"totalWebReadMcpCount"`
	TotalZreadMcpCount      int64              `json:"totalZreadMcpCount"`
	TotalSearchMcpCount     int64              `json:"totalSearchMcpCount"`
	ToolSummaryList         []ToolUsageSummary `json:"toolSummaryList"`
}

// ToolUsageSummary is one tool's window-wide total.
type ToolUsageSummary struct {
	ToolCode        string `json:"toolCode"`
	ToolName        string `json:"toolName"`
	ToolNameI18n    string `json:"toolNameI18n"`
	TotalUsageCount int64  `json:"totalUsageCount"`
	SortOrder       int    `json:"sortOrder"`
}

// ToolUsageSeries is one tool's per-bucket usage count across XTime.
type ToolUsageSeries struct {
	ToolCode        string  `json:"toolCode"`
	ToolName        string  `json:"toolName"`
	SortOrder       int     `json:"sortOrder"`
	UsageCount      []int64 `json:"usageCount"`
	TotalUsageCount int64   `json:"totalUsageCount"`
}

// GetQuotaLimit returns the plan tier and its quota windows.
func (s *QuotaService) GetQuotaLimit(ctx context.Context) (*QuotaLimitResponse, error) {
	return fetchEnvelope[QuotaData](ctx, s.client, s.request(QuotaLimitEndpoint), "quota limit")
}

// Monitor (quota/usage) endpoint paths under Region.MonitorBaseURL.
const (
	QuotaLimitEndpoint = "/usage/quota/limit"
	ModelUsageEndpoint = "/usage/model-usage"
	ToolUsageEndpoint  = "/usage/tool-usage"
)

// monitorUsageTimeFormat is the format the monitor API expects startTime/
// endTime query params in.
const monitorUsageTimeFormat = "2006-01-02 15:04:05"

// monitorUsagePath builds the endpoint + properly-encoded query string for a
// monitor usage request. startTime/endTime contain a space and colons,
// which must go through url.Values (not raw fmt.Sprintf string
// concatenation) — an unescaped space in the query string trips an HTTP/2
// stream error against this API.
//
// The times are formatted in serverTZ (the timezone the monitor API operates
// in) because the server reads these zoneless strings as its own wall-clock.
// Formatting in the viewer's local zone would request a shifted window
// (UTC+offset hours off the intended range). See Config.MonitorTimezone.
func monitorUsagePath(endpoint string, startTime, endTime time.Time, serverTZ *time.Location) string {
	q := url.Values{}
	q.Set("startTime", startTime.In(serverTZ).Format(monitorUsageTimeFormat))
	q.Set("endTime", endTime.In(serverTZ).Format(monitorUsageTimeFormat))
	return endpoint + "?" + q.Encode()
}

// GetModelUsage returns per-model token usage between startTime and endTime.
func (s *QuotaService) GetModelUsage(ctx context.Context, startTime, endTime time.Time) (*ModelUsageResponse, error) {
	path := monitorUsagePath(ModelUsageEndpoint, startTime, endTime, s.client.MonitorTimezone())
	return fetchEnvelope[ModelUsageData](ctx, s.client, s.request(path), "model usage")
}

// GetToolUsage returns MCP tool usage between startTime and endTime.
func (s *QuotaService) GetToolUsage(ctx context.Context, startTime, endTime time.Time) (*ToolUsageResponse, error) {
	path := monitorUsagePath(ToolUsageEndpoint, startTime, endTime, s.client.MonitorTimezone())
	return fetchEnvelope[ToolUsageData](ctx, s.client, s.request(path), "tool usage")
}

// request builds a GET against the region's monitor root.
func (s *QuotaService) request(path string) apiRequest {
	return apiRequest{method: "GET", baseURL: s.client.config.Region.MonitorBaseURL(), path: path, service: "quota"}
}
