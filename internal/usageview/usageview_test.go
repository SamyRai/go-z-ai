package usageview

import (
	"strings"
	"testing"
	"time"
)

func TestPaceOnTrack(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Hour)
	now := start.Add(5 * time.Hour) // 50% elapsed

	p, ok := Pace(0.40, start, end, now) // 40% used at 50% elapsed
	if !ok {
		t.Fatal("expected ok")
	}
	if p.ExhaustsEarly {
		t.Errorf("40%% used at 50%% elapsed should not exhaust early: %+v", p)
	}
	if p.Projected < 0.79 || p.Projected > 0.81 {
		t.Errorf("projected = %.3f, want ~0.80", p.Projected)
	}
	if s := FormatPace(p); !strings.Contains(s, "on track") {
		t.Errorf("FormatPace = %q, want 'on track'", s)
	}
}

func TestPaceExhaustsEarly(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Hour)
	now := start.Add(2 * time.Hour) // 20% elapsed

	p, ok := Pace(0.60, start, end, now) // 60% used at 20% elapsed → hot
	if !ok {
		t.Fatal("expected ok")
	}
	if !p.ExhaustsEarly {
		t.Fatalf("60%% used at 20%% elapsed should exhaust early: %+v", p)
	}
	// timeToExhaust = 2h / 0.6 = 3h20m from start; before = 10h - 3h20m = 6h40m.
	if p.ExhaustsBefore < 6*time.Hour+30*time.Minute || p.ExhaustsBefore > 6*time.Hour+50*time.Minute {
		t.Errorf("ExhaustsBefore = %s, want ~6h40m", p.ExhaustsBefore)
	}
	if s := FormatPace(p); !strings.Contains(s, "run out") || !strings.Contains(s, "before reset") {
		t.Errorf("FormatPace = %q, want a run-out warning", s)
	}
}

func TestPaceWindowJustStarted(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Hour)

	p, ok := Pace(0.0, start, end, start) // now == start
	if !ok {
		t.Fatal("expected ok")
	}
	if p.WindowElapsed != 0 {
		t.Errorf("WindowElapsed = %v, want 0", p.WindowElapsed)
	}
	if s := FormatPace(p); !strings.Contains(s, "just started") {
		t.Errorf("FormatPace = %q, want 'just started'", s)
	}
}

func TestPaceInvalidBounds(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	if _, ok := Pace(0.5, time.Time{}, start, start); ok {
		t.Error("zero start should be rejected")
	}
	if _, ok := Pace(0.5, start, start, start); ok {
		t.Error("zero-length window should be rejected")
	}
	if _, ok := Pace(0.5, start, start.Add(-time.Hour), start); ok {
		t.Error("inverted window should be rejected")
	}
}

// A tiny usage fraction over a long window must not falsely report early
// exhaustion — the naive elapsed/usedFraction duration would overflow negative.
func TestPaceTinyUsageNoFalseEarly(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour) // weekly window
	now := start.Add(24 * time.Hour)     // 1 day in

	p, ok := Pace(0.00001, start, end, now) // 0.001% used
	if !ok {
		t.Fatal("expected ok")
	}
	if p.ExhaustsEarly {
		t.Errorf("negligible usage must not exhaust early: %+v", p)
	}
	if p.ExhaustsBefore < 0 {
		t.Errorf("ExhaustsBefore must never be negative, got %s", p.ExhaustsBefore)
	}
}

func TestPaceClampsElapsed(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Hour)

	// now past the window end → elapsed clamps to 100%.
	p, ok := Pace(0.5, start, end, end.Add(3*time.Hour))
	if !ok {
		t.Fatal("expected ok")
	}
	if p.WindowElapsed != 1 {
		t.Errorf("WindowElapsed = %v, want 1 (clamped)", p.WindowElapsed)
	}
	if p.ExhaustsEarly {
		t.Errorf("50%% used at 100%% elapsed should not be early: %+v", p)
	}
}

func TestCompactDuration(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second:             "under 1m",
		45 * time.Minute:             "45m",
		2 * time.Hour:                "2h",
		2*time.Hour + 30*time.Minute: "2h 30m",
		3 * 24 * time.Hour:           "3d",
		2*24*time.Hour + 5*time.Hour: "2d",
	}
	for d, want := range cases {
		if got := compactDuration(d); got != want {
			t.Errorf("compactDuration(%s) = %q, want %q", d, got, want)
		}
	}
}

// ParseMonitorTime interprets a zoneless monitor label in the given server
// zone, yielding the correct absolute instant. "09:00 CST" is 01:00 UTC.
func TestParseMonitorTime(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	got, err := ParseMonitorTime("2026-08-02 09:00", cst)
	if err != nil {
		t.Fatalf("ParseMonitorTime: %v", err)
	}
	want := time.Date(2026, 8, 2, 1, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseMonitorTime = %s (UTC %s), want %s", got, got.UTC(), want)
	}

	// Empty → zero, no error.
	if t2, err := ParseMonitorTime("", cst); err != nil || !t2.IsZero() {
		t.Errorf("ParseMonitorTime(\"\") = %v, %v, want zero/nil", t2, err)
	}
	// Unparseable → error.
	if _, err := ParseMonitorTime("garbage", cst); err == nil {
		t.Error("ParseMonitorTime(garbage): want error")
	}
	// Also accepts the finer :05 layout (the query-param format).
	if _, err := ParseMonitorTime("2026-08-02 09:00:05", cst); err != nil {
		t.Errorf("ParseMonitorTime(:05 layout): %v", err)
	}
}

// LocalizeXTime converts each server-local label into the viewer's local
// display string; a nil server zone leaves labels untouched.
func TestLocalizeXTime(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	// Pin time.Local for the duration of this test so the expected output is
	// deterministic regardless of where the test runs.
	orig := time.Local
	time.Local = time.FixedZone("TEST+0", 0)
	defer func() { time.Local = orig }()

	labels := []string{"2026-08-02 09:00", "2026-08-02 10:00"}
	got := LocalizeXTime(labels, cst)
	want := []string{"2026-08-02 01:00", "2026-08-02 02:00"} // 09:00 CST = 01:00 UTC
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("LocalizeXTime[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// nil server zone → labels pass through unchanged.
	pass := LocalizeXTime(labels, nil)
	for i := range pass {
		if pass[i] != labels[i] {
			t.Errorf("LocalizeXTime(nil)[%d] = %q, want passthrough %q", i, pass[i], labels[i])
		}
	}

	// A malformed label passes through rather than blanking the row.
	mixed := LocalizeXTime([]string{"2026-08-02 09:00", "garbage"}, cst)
	if mixed[1] != "garbage" {
		t.Errorf("LocalizeXTime malformed passthrough = %q, want %q", mixed[1], "garbage")
	}
}

// ZoneNote is empty when the server zone matches the viewer's local offset, and
// names both zones when they differ.
func TestZoneNoteDiffer(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("TEST+0", 0) // viewer at UTC+0
	defer func() { time.Local = orig }()

	cst := time.FixedZone("CST", 8*3600)
	note := ZoneNote(cst)
	if note == "" {
		t.Fatal("ZoneNote should be non-empty when zones differ")
	}
	if !strings.Contains(note, "local") || !strings.Contains(note, "CST") || !strings.Contains(note, "+08:00") {
		t.Errorf("ZoneNote = %q, want it to mention local, CST, and +08:00", note)
	}
}

// When the viewer is already in the server's offset, ZoneNote is empty.
func TestZoneNoteSame(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("CST", 8*3600) // viewer at UTC+8, same as server
	defer func() { time.Local = orig }()

	if note := ZoneNote(time.FixedZone("CST", 8*3600)); note != "" {
		t.Errorf("ZoneNote matching zones = %q, want \"\"", note)
	}
	if note := ZoneNote(nil); note != "" {
		t.Errorf("ZoneNote(nil) = %q, want \"\"", note)
	}
}
