package usageview

import (
	"fmt"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// LimitSummary is the display form of one quota window. The CLI and the TUI
// both render quota through it, so they can't drift apart.
type LimitSummary struct {
	Title string  // e.g. "5-hour credit window"
	Used  float64 // fraction consumed, 0..1+
	// Counts is "1.4K / 12.0K used — 10.6K left" when the server reports
	// absolute amounts; empty for percentage-only windows.
	Counts string
	// Reset is "resets 15:04 MST (in 2h 3m)", or a note that the window
	// starts with the next request.
	Reset string
	// Pace and Peak are optional advisory lines for model-usage windows.
	Pace, Peak string
	// Tools breaks an MCP-call lane down by tool.
	Tools []string
}

// SummarizeLimit describes l at now, with times in loc (the viewer's zone).
func SummarizeLimit(l client.QuotaLimit, mode BillingMode, now time.Time, loc *time.Location) LimitSummary {
	if loc == nil {
		loc = time.Local
	}
	s := LimitSummary{Title: l.WindowDescription(), Used: l.UsedFraction()}
	if l.Usage > 0 {
		s.Counts = fmt.Sprintf("%s / %s used — %s left",
			FormatCount(int64(l.CurrentValue)), FormatCount(int64(l.Usage)), FormatCount(int64(l.Remaining)))
	}
	if reset := l.ResetTime(); reset.IsZero() {
		s.Reset = "window starts with the next request"
	} else {
		s.Reset = fmt.Sprintf("resets %s (in %s)", reset.In(loc).Format("Jan 2 15:04 MST"), FormatDuration(max(reset.Sub(now), 0)))
	}
	if l.IsModelLimit() {
		if start := l.WindowStart(); !start.IsZero() {
			if pace, ok := Pace(s.Used, start, l.ResetTime(), now); ok {
				s.Pace = FormatPace(pace)
			}
		}
		s.Peak = FormatPeakWarning(now, mode, loc)
	}
	for _, d := range l.UsageDetails {
		name := d.DisplayName
		if name == "" {
			name = d.ModelCode
		}
		s.Tools = append(s.Tools, fmt.Sprintf("%s: %d", name, d.Usage))
	}
	return s
}
