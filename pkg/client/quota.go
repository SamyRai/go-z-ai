package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Quota limit type constants
const (
	QuotaTypeTokensLimit = "TOKENS_LIMIT" // API request token limits
	QuotaTypeTimeLimit   = "TIME_LIMIT"   // MCP tools usage limits
)

// Time unit code constants from Z.ai API
const (
	UnitCodeHourly  = 3 // 5-hour rolling window (for TOKENS_LIMIT)
	UnitCodeWeekly  = 6 // Weekly rolling window (for TOKENS_LIMIT)
	UnitCodeMonthly = 5 // Monthly quota (for TIME_LIMIT - MCP tools)
)

// QuotaWindowConfig defines a quota window configuration with human-readable metadata
type QuotaWindowConfig struct {
	Type        string // Limit type (TOKENS_LIMIT, TIME_LIMIT)
	UnitCode    int    // API unit code
	Number      int    // Number of units
	Description string // Human-readable description
}

// Known quota window configurations
// These map the cryptic API unit/number codes to understandable window descriptions
// When adding support for new quota window types, add them here and update the guide
var quotaWindowConfigs = []QuotaWindowConfig{
	// Token limit windows (API request quotas)
	{Type: QuotaTypeTokensLimit, UnitCode: UnitCodeHourly, Number: 5, Description: "5-hour rolling token window"},
	{Type: QuotaTypeTokensLimit, UnitCode: UnitCodeWeekly, Number: 1, Description: "weekly token window"},

	// MCP tools limit windows (external tool usage quotas)
	{Type: QuotaTypeTimeLimit, UnitCode: UnitCodeMonthly, Number: 1, Description: "monthly MCP tools quota"},
}

// findWindowConfig looks up a quota window configuration by type, unit code, and number
// Returns the matching config or a generic fallback if no match is found
// This provides a centralized mapping for all known quota window types
func findWindowConfig(limitType string, unitCode, number int) QuotaWindowConfig {
	for _, config := range quotaWindowConfigs {
		if config.Type == limitType && config.UnitCode == unitCode && config.Number == number {
			return config
		}
	}

	// Return a generic fallback config for unknown window types
	// This ensures we always return some description even for new/unknown quota types
	return QuotaWindowConfig{
		Type:        limitType,
		UnitCode:    unitCode,
		Number:      number,
		Description: fmt.Sprintf("unknown %s window (unit %d × %d)", limitType, unitCode, number),
	}
}

// QuotaService handles quota and usage monitoring
type QuotaService struct {
	client *Client
}

// QuotaLimitResponse represents the full quota limit API response
type QuotaLimitResponse struct {
	Code    int       `json:"code"`
	Msg     string    `json:"msg"`
	Data    QuotaData `json:"data"`
	Success bool      `json:"success"`
}

// QuotaData contains the quota data
type QuotaData struct {
	Limits []QuotaLimit `json:"limits"`
	Level  string       `json:"level"` // pro, lite, max
}

// QuotaLimit represents individual quota limits from the Z.ai API
//
// The Z.ai API returns quota limits with cryptic unit/number codes that need
// to be mapped to human-readable window descriptions. Use WindowDescription()
// to get clear descriptions like "5-hour rolling token window" instead of
// raw codes like "TOKENS_LIMIT (unit 3 × 5)".
//
// Field meanings:
// - Type: QuotaTypeTokensLimit (API calls) or QuotaTypeTimeLimit (MCP tools)
// - Unit: Time unit code (UnitCodeHourly=3, UnitCodeWeekly=6, UnitCodeMonthly=5)
// - Number: Number of time units (e.g., 5 for 5-hour, 1 for weekly/monthly)
// - Usage: Total limit (0 = no limit provided by API)
// - CurrentValue: Current usage count
// - Remaining: Remaining quota
// - Percentage: Usage percentage (0-100)
// - NextResetTime: Next reset timestamp (milliseconds since epoch)
// - UsageDetails: Tool-specific breakdown for TIME_LIMIT quotas
type QuotaLimit struct {
	Type          string            `json:"type"`                   // Quota type: TOKENS_LIMIT or TIME_LIMIT
	Unit          int               `json:"unit"`                   // Time unit code: 3, 5, or 6
	Number        int               `json:"number"`                 // Number of time units
	Usage         int               `json:"usage"`                  // Total usage limit (0 = no limit provided)
	CurrentValue  int               `json:"currentValue"`           // Current usage count
	Remaining     int               `json:"remaining"`              // Remaining quota
	Percentage    float64           `json:"percentage"`             // Usage percentage (0-100)
	NextResetTime int64             `json:"nextResetTime"`          // Next reset timestamp (milliseconds since epoch)
	UsageDetails  []ToolUsageDetail `json:"usageDetails,omitempty"` // Tool-specific breakdown for TIME_LIMIT
}

// ToolUsageDetail represents tool usage breakdown for TIME_LIMIT quotas
// ModelCode identifies which tool: "search-prime", "web-reader", "zread"
type ToolUsageDetail struct {
	ModelCode string `json:"modelCode"`
	Usage     int    `json:"usage"`
}

// WindowDescription returns a human-readable description of the quota window
// by looking up the configuration in the known quota windows mapping.
// This provides clear descriptions like "5-hour rolling token window" instead
// of cryptic API codes like "TOKENS_LIMIT (unit 3 × 5)".
//
// Example usage:
//
//	limit.WindowDescription() // "5-hour rolling token window"
func (q *QuotaLimit) WindowDescription() string {
	config := findWindowConfig(q.Type, q.Unit, q.Number)
	return config.Description
}

// IsTokenLimit reports whether this is a TOKENS_LIMIT (API request quota)
func (q *QuotaLimit) IsTokenLimit() bool {
	return q.Type == QuotaTypeTokensLimit
}

// IsToolsLimit reports whether this is a TIME_LIMIT (MCP tools quota)
func (q *QuotaLimit) IsToolsLimit() bool {
	return q.Type == QuotaTypeTimeLimit
}

// IsExhausted reports whether the quota window is at or near 100% usage
func (q *QuotaLimit) IsExhausted() bool {
	return q.Percentage >= 99.0
}

// IsLow reports whether the quota window is below 20% remaining
func (q *QuotaLimit) IsLow() bool {
	return q.Remaining > 0 && q.Percentage >= 80.0
}

// ResetTime returns the next reset time as a time.Time
func (q *QuotaLimit) ResetTime() time.Time {
	if q.NextResetTime == 0 {
		return time.Time{}
	}
	return time.UnixMilli(q.NextResetTime)
}

// WindowDuration returns the length of this quota's rolling window (5 hours,
// one week, ...), derived from the Unit/Number codes. Returns 0 for an unknown
// unit code, in which case the window start can't be computed.
func (q *QuotaLimit) WindowDuration() time.Duration {
	var unit time.Duration
	switch q.Unit {
	case UnitCodeHourly:
		unit = time.Hour
	case UnitCodeWeekly:
		unit = 7 * 24 * time.Hour
	case UnitCodeMonthly:
		unit = 30 * 24 * time.Hour
	default:
		return 0
	}
	n := q.Number
	if n <= 0 {
		n = 1
	}
	return time.Duration(n) * unit
}

// WindowStart returns when the current rolling window began, i.e. the reset
// time minus the window duration. Returns the zero time when either the reset
// time or the window duration is unknown.
func (q *QuotaLimit) WindowStart() time.Time {
	reset := q.ResetTime()
	d := q.WindowDuration()
	if reset.IsZero() || d == 0 {
		return time.Time{}
	}
	return reset.Add(-d)
}

// ModelUsageResponse represents model usage statistics for a time window,
// bucketed daily or hourly depending on the requested range (see
// ModelUsageData.Granularity). Verified against the live API — the response
// is a time-series object, not a flat list.
type ModelUsageResponse struct {
	Code    int            `json:"code"`
	Msg     string         `json:"msg"`
	Data    ModelUsageData `json:"data"`
	Success bool           `json:"success"`
}

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
type ToolUsageResponse struct {
	Code    int           `json:"code"`
	Msg     string        `json:"msg"`
	Data    ToolUsageData `json:"data"`
	Success bool          `json:"success"`
}

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

// NewQuotaService creates a new quota service
func NewQuotaService(client *Client) *QuotaService {
	return &QuotaService{client: client}
}

// GetQuotaLimit retrieves the current quota limit status
func (s *QuotaService) GetQuotaLimit(ctx context.Context) (*QuotaLimitResponse, error) {
	var result QuotaLimitResponse
	if err := s.client.doRequestBase(ctx, s.client.config.Region.monitorBaseURL(), "GET", QuotaLimitEndpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get quota limit: %w", err)
	}
	return &result, nil
}

// monitorUsageTimeFormat is the format the monitor API expects startTime/
// endTime query params in.
const monitorUsageTimeFormat = "2006-01-02 15:04:05"

// monitorUsagePath builds the endpoint + properly-encoded query string for a
// monitor usage request. startTime/endTime contain a space and colons,
// which must go through url.Values (not raw fmt.Sprintf string
// concatenation) — an unescaped space in the query string trips an HTTP/2
// stream error against this API.
func monitorUsagePath(endpoint string, startTime, endTime time.Time) string {
	q := url.Values{}
	q.Set("startTime", startTime.Format(monitorUsageTimeFormat))
	q.Set("endTime", endTime.Format(monitorUsageTimeFormat))
	return endpoint + "?" + q.Encode()
}

// GetModelUsage retrieves model usage statistics for a time window
func (s *QuotaService) GetModelUsage(ctx context.Context, startTime, endTime time.Time) (*ModelUsageResponse, error) {
	var result ModelUsageResponse
	path := monitorUsagePath(ModelUsageEndpoint, startTime, endTime)
	if err := s.client.doRequestBase(ctx, s.client.config.Region.monitorBaseURL(), "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	return &result, nil
}

// GetToolUsage retrieves MCP tool usage statistics for a time window
func (s *QuotaService) GetToolUsage(ctx context.Context, startTime, endTime time.Time) (*ToolUsageResponse, error) {
	var result ToolUsageResponse
	path := monitorUsagePath(ToolUsageEndpoint, startTime, endTime)
	if err := s.client.doRequestBase(ctx, s.client.config.Region.monitorBaseURL(), "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get tool usage: %w", err)
	}
	return &result, nil
}
