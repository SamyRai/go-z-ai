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

// monitorUsagePath must format the query window in the server's timezone, not
// the time value's own zone — the monitor API reads these zoneless strings as
// its own wall-clock. A UTC instant must therefore appear as its UTC+8
// equivalent in the query string, regardless of the input time's Location.
func TestMonitorUsagePathServerTimezone(t *testing.T) {
	// A single absolute instant: 2026-08-02 01:00:00 UTC.
	utcInstant := time.Date(2026, 8, 2, 1, 0, 0, 0, time.UTC)
	// The same instant expressed in two different zones — the wire format
	// must be identical because both resolve to the same server-zone string.
	cestInstant := utcInstant.In(time.FixedZone("CEST", 2*3600))

	cst := MonitorServerTZ // UTC+8
	want := "/usage/model-usage?endTime=2026-08-02+09%3A00%3A00&startTime=2026-08-02+09%3A00%3A00"

	for name, in := range map[string]time.Time{"utc": utcInstant, "cest": cestInstant} {
		got := monitorUsagePath(ModelUsageEndpoint, in, in, cst)
		if got != want {
			t.Errorf("%s: monitorUsagePath = %q, want %q", name, got, want)
		}
	}
}

// When a Config.MonitorTimezone override is set, it must win over the region
// default for query formatting and for the Client.MonitorTimezone() accessor.
func TestMonitorTimezoneOverride(t *testing.T) {
	override := time.FixedZone("OVERRIDE", -5*3600)
	c := newTestClient(t, "http://example.test", Config{MonitorTimezone: override})
	if got := c.MonitorTimezone(); got != override {
		t.Errorf("MonitorTimezone() = %v, want override %v", got, override)
	}
	// Without override, it falls back to the region default (CST/UTC+8).
	c2 := newTestClient(t, "http://example.test", Config{})
	if got := c2.MonitorTimezone(); got.String() != MonitorServerTZ.String() {
		t.Errorf("default MonitorTimezone() = %v, want %v", got, MonitorServerTZ)
	}
}
