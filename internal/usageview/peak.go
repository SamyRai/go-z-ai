package usageview

import (
	"fmt"
	"time"
)

// Z.AI applies a peak-hours surcharge on the GLM Coding Plan: high-tier
// tokens (GLM-5.x / Claude-class) count at 3× against the quota during peak
// and 2× otherwise. The window is weekdays 14:00–18:00 in the server's
// timezone (UTC+8 / CST), for all users globally — it is NOT shifted to the
// viewer's local timezone.
//
// Source: docs.bigmodel.cn/cn/coding-plan/team (verified 2026-08-06).
// The quota API does NOT expose the window or multiplier — they are applied
// server-side, so the already-reported Percentage/CurrentValue reflect the
// surcharge but the timeframe itself must be known client-side to warn about
// it. Update these constants if Z.AI changes the window.
const (
	peakStartHour = 14
	peakEndHour   = 18
)

// IsPeak reports whether the given instant, interpreted in serverTZ, falls in
// Z.AI's weekday peak-hours window (14:00–18:00 server-local). Weekend days
// are never peak. serverTZ is the monitor server's timezone (CST / UTC+8),
// available via client.MonitorTimezone().
func IsPeak(now time.Time, serverTZ *time.Location) bool {
	if serverTZ == nil {
		return false
	}
	t := now.In(serverTZ)
	if d := t.Weekday(); d == time.Saturday || d == time.Sunday {
		return false
	}
	return t.Hour() >= peakStartHour && t.Hour() < peakEndHour
}

// PeakEndsAt returns when the current peak window ends (18:00 server-local),
// or the zero time if now is not in peak.
func PeakEndsAt(now time.Time, serverTZ *time.Location) time.Time {
	if !IsPeak(now, serverTZ) {
		return time.Time{}
	}
	t := now.In(serverTZ)
	return time.Date(t.Year(), t.Month(), t.Day(), peakEndHour, 0, 0, 0, serverTZ)
}

// FormatPeakWarning returns a one-line peak-hours notice when now is in the
// peak window, "" otherwise. Example:
//
//	⚡ peak hours: tokens count 3× until 18:00 CST
//
// The caller is responsible for styling (color/bold); this returns plain text.
func FormatPeakWarning(now time.Time, serverTZ *time.Location) string {
	end := PeakEndsAt(now, serverTZ)
	if end.IsZero() {
		return ""
	}
	return fmt.Sprintf("⚡ peak hours: tokens count 3× until %s", end.Format("15:04 MST"))
}
