package usageview

import (
	"strings"
	"testing"
	"time"
)

// CST is the server timezone Z.AI's monitor operates in (UTC+8).
var cst = time.FixedZone("CST", 8*3600)

func TestIsPeak(t *testing.T) {
	// A Wednesday in the peak window.
	wedInPeak := time.Date(2026, 8, 5, 15, 30, 0, 0, cst)
	// A Wednesday just before peak.
	wedBeforePeak := time.Date(2026, 8, 5, 13, 59, 0, 0, cst)
	// A Wednesday at the peak boundary (18:00 = exclusive end).
	wedAtEnd := time.Date(2026, 8, 5, 18, 0, 0, 0, cst)
	// A Wednesday evening (off-peak).
	wedEvening := time.Date(2026, 8, 5, 22, 0, 0, 0, cst)
	// A Saturday during peak hours (weekends are never peak).
	satInHours := time.Date(2026, 8, 8, 15, 30, 0, 0, cst)
	// A Sunday during peak hours.
	sunInHours := time.Date(2026, 8, 9, 16, 0, 0, 0, cst)
	// Monday at 14:00 (inclusive start).
	monAtStart := time.Date(2026, 8, 3, 14, 0, 0, 0, cst)

	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"Wednesday mid-peak", wedInPeak, true},
		{"Wednesday just before 14:00", wedBeforePeak, false},
		{"Wednesday at 18:00 (end exclusive)", wedAtEnd, false},
		{"Wednesday evening", wedEvening, false},
		{"Saturday in hours (weekend)", satInHours, false},
		{"Sunday in hours (weekend)", sunInHours, false},
		{"Monday at 14:00 (start inclusive)", monAtStart, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsPeak(c.now, cst); got != c.want {
				t.Errorf("IsPeak = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsPeakViewerTimezoneIrrelevant(t *testing.T) {
	// The same instant expressed in different viewer timezones must give the
	// same result — peak is defined in server time (CST), not the viewer's.
	instant := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC) // 17:30 CST = peak
	ceST := time.FixedZone("CEST", 2*3600)                  // viewer in Europe
	baseResult := IsPeak(instant, cst)
	viewerResult := IsPeak(instant.In(ceST), cst)
	if baseResult != viewerResult {
		t.Errorf("peak detection changed with viewer tz: base=%v viewer=%v", baseResult, viewerResult)
	}
	if !baseResult {
		t.Error("09:30 UTC = 17:30 CST should be peak")
	}
}

func TestIsPeakNilTimezone(t *testing.T) {
	if IsPeak(time.Now(), nil) {
		t.Error("IsPeak with nil serverTZ should be false")
	}
}

func TestPeakEndsAt(t *testing.T) {
	inPeak := time.Date(2026, 8, 5, 15, 30, 0, 0, cst)
	end := PeakEndsAt(inPeak, cst)
	want := time.Date(2026, 8, 5, 18, 0, 0, 0, cst)
	if !end.Equal(want) {
		t.Errorf("PeakEndsAt = %s, want %s", end, want)
	}

	offPeak := time.Date(2026, 8, 5, 22, 0, 0, 0, cst)
	if end := PeakEndsAt(offPeak, cst); !end.IsZero() {
		t.Errorf("PeakEndsAt off-peak should be zero, got %s", end)
	}
}

func TestFormatPeakWarning(t *testing.T) {
	inPeak := time.Date(2026, 8, 5, 15, 30, 0, 0, cst)
	got := FormatPeakWarning(inPeak, cst)
	if got == "" {
		t.Fatal("expected a warning during peak hours")
	}
	if !strings.Contains(got, "3×") {
		t.Errorf("warning should mention 3× multiplier: %q", got)
	}
	if !strings.Contains(got, "18:00") {
		t.Errorf("warning should mention end time 18:00: %q", got)
	}
	if !strings.Contains(got, "CST") {
		t.Errorf("warning should mention the server timezone CST: %q", got)
	}

	offPeak := time.Date(2026, 8, 5, 22, 0, 0, 0, cst)
	if got := FormatPeakWarning(offPeak, cst); got != "" {
		t.Errorf("FormatPeakWarning off-peak should be empty, got %q", got)
	}
}
