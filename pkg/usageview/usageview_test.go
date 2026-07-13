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
