// Package usageview holds pure, presentation-only helpers for rendering
// usage/quota data (time windows, relative timestamps, compact counters, and
// a density-heatmap ramp). It has no CLI or TUI dependency so both the
// Cobra "accounts usage"/"usage" commands and the TUI's Usage tab render the
// identical output — the two must never drift.
package usageview

import (
	"fmt"
	"time"
)

// Window returns the [start, end) time range covering the last days days
// (or just today, if today is true or days<=1).
func Window(days int, today bool) (time.Time, time.Time) {
	now := time.Now()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	if today || days <= 1 {
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return startOfToday, endOfToday
	}

	startDay := endOfToday.AddDate(0, 0, -(days - 1))
	start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, startDay.Location())
	return start, endOfToday
}

// HeatmapLevels is the density ramp used to render each bucket, from empty
// to peak.
var HeatmapLevels = []rune(" ░▒▓█")

// HeatmapBlocks renders one character per value, scaled against that row's
// own maximum (not a global maximum) so each series' own activity pattern
// stays readable regardless of how it compares in magnitude to others.
func HeatmapBlocks(values []int64) string {
	var max int64
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	runes := make([]rune, len(values))
	for i, v := range values {
		if max == 0 || v == 0 {
			runes[i] = HeatmapLevels[0]
			continue
		}
		level := int(float64(v)/float64(max)*float64(len(HeatmapLevels)-1) + 0.5)
		if level < 1 {
			level = 1
		}
		if level >= len(HeatmapLevels) {
			level = len(HeatmapLevels) - 1
		}
		runes[i] = HeatmapLevels[level]
	}
	return string(runes)
}

// HeatmapLevelIndex returns the HeatmapLevels index HeatmapBlocks would have
// picked for a single value against max, so callers that want to recolor
// each character (e.g. the TUI) don't have to reimplement the scaling.
func HeatmapLevelIndex(v, max int64) int {
	if max == 0 || v == 0 {
		return 0
	}
	level := int(float64(v)/float64(max)*float64(len(HeatmapLevels)-1) + 0.5)
	if level < 1 {
		level = 1
	}
	if level >= len(HeatmapLevels) {
		level = len(HeatmapLevels) - 1
	}
	return level
}

// FormatRelativeTime renders t as a short relative duration ("3m ago"), or
// "never" for the zero value.
func FormatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// QuotaPace describes how fast a rolling quota window is being consumed
// relative to how much of the window has elapsed — the "am I burning too fast?"
// question behind the common complaint that Coding Plan limits run out sooner
// than expected. It is a straight-line extrapolation of the window's own
// reported usage; it makes no assumption about peak/off-peak pricing.
type QuotaPace struct {
	WindowElapsed  float64       // fraction of the window elapsed, [0,1]
	Used           float64       // fraction of the quota consumed, [0,1+]
	Projected      float64       // usage extrapolated to the window end at the current pace
	ExhaustsEarly  bool          // true when the quota is projected to run out before reset
	ExhaustsBefore time.Duration // how long before reset it runs out (only when ExhaustsEarly)
}

// Pace projects a rolling window's consumption to its reset. usedFraction is
// the window's reported usage in [0,1] (e.g. Percentage/100). It returns false
// when the window bounds are unusable (zero/inverted), so callers can skip the
// line entirely.
func Pace(usedFraction float64, windowStart, windowEnd, now time.Time) (QuotaPace, bool) {
	total := windowEnd.Sub(windowStart)
	if windowStart.IsZero() || windowEnd.IsZero() || total <= 0 {
		return QuotaPace{}, false
	}

	elapsed := now.Sub(windowStart)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > total {
		elapsed = total
	}

	p := QuotaPace{
		WindowElapsed: float64(elapsed) / float64(total),
		Used:          usedFraction,
	}
	if elapsed <= 0 {
		return p, true // no time elapsed yet — nothing to project
	}

	p.Projected = usedFraction / p.WindowElapsed
	// The window runs out early exactly when the projected end-of-window usage
	// exceeds 100%. Deriving the lead time as a fraction of the window (rather
	// than elapsed/usedFraction) avoids a time.Duration overflow when
	// usedFraction is tiny.
	if p.Projected > 1 {
		p.ExhaustsEarly = true
		// Usage hits 100% at elapsed-fraction WindowElapsed/usedFraction of the
		// window; the remainder is how long before reset it runs out.
		p.ExhaustsBefore = time.Duration((1 - p.WindowElapsed/usedFraction) * float64(total))
	}
	return p, true
}

// FormatPace renders a QuotaPace as a single compact line.
func FormatPace(p QuotaPace) string {
	if p.WindowElapsed <= 0 {
		return "— (window just started)"
	}
	head := fmt.Sprintf("%.0f%% used at %.0f%% of window elapsed", p.Used*100, p.WindowElapsed*100)
	switch {
	case p.ExhaustsEarly:
		return head + fmt.Sprintf(" — on pace to run out ~%s before reset", compactDuration(p.ExhaustsBefore))
	case p.Projected > 0:
		return head + fmt.Sprintf(" — on track (~%.0f%% projected by reset)", p.Projected*100)
	default:
		return head
	}
}

// compactDuration renders a positive duration as "3d", "5h", "2h 30m", or
// "45m" — coarse enough for a pace hint, never seconds.
func compactDuration(d time.Duration) string {
	if d < time.Minute {
		return "under 1m"
	}
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d >= time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// FormatCount renders large counters compactly (e.g. 159454762 -> "159.5M").
func FormatCount(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
