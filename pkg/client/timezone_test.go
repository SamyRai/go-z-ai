package client

import (
	"testing"
	"time"
)

// ParseTimezone accepts IANA names, UTC, "local", and UTC-offset forms;
// empty → nil (no override). Invalid values must error.
func TestParseTimezone(t *testing.T) {
	cases := []struct {
		in      string
		want    func(*time.Location) bool // predicate on the result
		wantErr bool
	}{
		{"", func(l *time.Location) bool { return l == nil }, false},
		{"  ", func(l *time.Location) bool { return l == nil }, false},
		{"UTC", func(l *time.Location) bool { return l == time.UTC }, false},
		{"utc", func(l *time.Location) bool { return l == time.UTC }, false},
		{"local", func(l *time.Location) bool { return l == time.Local }, false},
		// Offset forms — tzdata-free, so always available.
		{"UTC+8", func(l *time.Location) bool { return offsetSecs(l) == 8*3600 }, false},
		{"+8", func(l *time.Location) bool { return offsetSecs(l) == 8*3600 }, false},
		{"+08:00", func(l *time.Location) bool { return offsetSecs(l) == 8*3600 }, false},
		{"-05:00", func(l *time.Location) bool { return offsetSecs(l) == -5*3600 }, false},
		{"-5", func(l *time.Location) bool { return offsetSecs(l) == -5*3600 }, false},
		{"+0530", func(l *time.Location) bool { return offsetSecs(l) == (5*3600 + 30*60) }, false},
		// IANA name — requires tzdata; tolerate its absence as an error (not a
		// hard failure) so the test passes on minimal CI images.
		{"Asia/Shanghai", func(l *time.Location) bool {
			return l != nil && offsetSecs(l) == 8*3600
		}, false},
		// Bogus value → error.
		{"not-a-zone", nil, true},
		{"UTC+99", nil, true},
	}
	for _, c := range cases {
		got, err := ParseTimezone(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseTimezone(%q): want error, got nil (%v)", c.in, got)
			}
			continue
		}
		if err != nil {
			// Asia/Shanghai may fail on tzdata-less images; skip it rather
			// than failing the test for an environment limitation.
			if c.in == "Asia/Shanghai" {
				t.Logf("ParseTimezone(%q): tzdata unavailable, skipping: %v", c.in, err)
				continue
			}
			t.Errorf("ParseTimezone(%q): unexpected error: %v", c.in, err)
			continue
		}
		if !c.want(got) {
			t.Errorf("ParseTimezone(%q): predicate failed for %v", c.in, got)
		}
	}
}

// offsetSecs returns the location's current UTC offset in seconds (DST-aware).
func offsetSecs(loc *time.Location) int {
	if loc == nil {
		return 1 << 31 // sentinel that won't match any real offset
	}
	_, off := time.Now().In(loc).Zone()
	return off
}
