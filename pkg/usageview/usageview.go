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
