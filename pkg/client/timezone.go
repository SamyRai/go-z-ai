package client

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MonitorServerTZ is the timezone the Z.AI monitor API operates in. The
// monitor endpoints exchange zoneless time strings ("2006-01-02 15:04:05")
// for startTime/endTime query params and x_time bucket labels; the server
// reads and emits them in its own wall-clock, which (verified live against
// the global host, 2026-08-02) is China Standard Time (UTC+8). The China
// gateway is the same provider and assumed identical. A time.FixedZone is
// used instead of time.LoadLocation("Asia/Shanghai") so the value is correct
// without a tzdata dependency (Windows CI builds may ship without it).
// Override per client via Config.MonitorTimezone.
var MonitorServerTZ = time.FixedZone("CST", 8*3600)

// ParseTimezone parses a timezone string as accepted by ZAI_MONITOR_TIMEZONE:
// an IANA name ("Asia/Shanghai"), a "UTC" literal, a UTC offset ("UTC+8",
// "+08:00", "-5"), or "local". It returns nil (no error) for an empty string
// so callers can pass an unset env var through as "no override". Offsets and
// "UTC" need no tzdata; IANA names do.
func ParseTimezone(s string) (*time.Location, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	switch strings.ToLower(s) {
	case "local":
		return time.Local, nil
	case "utc":
		return time.UTC, nil
	}
	// Offset forms: "UTC+8", "UTC-05:00", "+8", "-05:00". Strip a "UTC" prefix.
	off := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "UTC"), "utc"))
	if off != "" && (off[0] == '+' || off[0] == '-') {
		loc, err := parseUTCOffset(off)
		if err != nil {
			return nil, fmt.Errorf("timezone %q: %w", s, err)
		}
		return loc, nil
	}
	// IANA name. Requires tzdata — surface a missing-tzdata error so the
	// caller can drop the override rather than silently ignoring it.
	loc, err := time.LoadLocation(s)
	if err != nil {
		return nil, fmt.Errorf("timezone %q: %w", s, err)
	}
	return loc, nil
}

// parseUTCOffset converts an offset like "+8", "-05:00", or "+0830" into a
// named fixed zone ("UTC+08:00"). It is the tzdata-free path of ParseTimezone.
func parseUTCOffset(s string) (*time.Location, error) {
	sign := s[0]
	body := strings.ReplaceAll(s[1:], ":", "")
	switch len(body) {
	case 1, 2: // hours only
		body = fmt.Sprintf("%02s", body) + "00"
	case 3: // hmm
		body = "0" + body
	case 4: // hhmm
	default:
		return nil, fmt.Errorf("invalid UTC offset %q", s)
	}
	h, err := strconv.Atoi(body[:2])
	if err != nil || h > 23 {
		return nil, fmt.Errorf("invalid UTC offset hours %q", s)
	}
	m, err := strconv.Atoi(body[2:])
	if err != nil || m > 59 {
		return nil, fmt.Errorf("invalid UTC offset minutes %q", s)
	}
	secs := h*3600 + m*60
	if sign == '-' {
		secs = -secs
	}
	return time.FixedZone(fmt.Sprintf("UTC%c%02d:%02d", sign, h, m), secs), nil
}
