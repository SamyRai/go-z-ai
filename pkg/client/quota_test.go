package client

import (
	"testing"
	"time"
)

func TestQuotaLimitWindowDuration(t *testing.T) {
	cases := []struct {
		name string
		unit int
		num  int
		want time.Duration
	}{
		{"5-hour", UnitCodeHourly, 5, 5 * time.Hour},
		{"weekly", UnitCodeWeekly, 1, 7 * 24 * time.Hour},
		{"monthly", UnitCodeMonthly, 1, 30 * 24 * time.Hour},
		{"number defaults to 1", UnitCodeHourly, 0, time.Hour},
		{"unknown unit", 99, 5, 0},
	}
	for _, c := range cases {
		q := QuotaLimit{Unit: c.unit, Number: c.num}
		if got := q.WindowDuration(); got != c.want {
			t.Errorf("%s: WindowDuration() = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestQuotaLimitWindowStart(t *testing.T) {
	reset := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	// Known 5-hour window: start is reset minus 5h.
	q := QuotaLimit{Unit: UnitCodeHourly, Number: 5, NextResetTime: reset.UnixMilli()}
	if got := q.WindowStart(); !got.Equal(reset.Add(-5 * time.Hour)) {
		t.Errorf("WindowStart() = %s, want %s", got, reset.Add(-5*time.Hour))
	}

	// No reset time → zero.
	noReset := QuotaLimit{Unit: UnitCodeHourly, Number: 5}
	if got := noReset.WindowStart(); !got.IsZero() {
		t.Errorf("WindowStart() with no reset = %s, want zero", got)
	}

	// Unknown unit (duration 0) → zero even with a reset time.
	unknownUnit := QuotaLimit{Unit: 99, Number: 5, NextResetTime: reset.UnixMilli()}
	if got := unknownUnit.WindowStart(); !got.IsZero() {
		t.Errorf("WindowStart() with unknown unit = %s, want zero", got)
	}
}
