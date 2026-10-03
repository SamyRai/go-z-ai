package usageview

import (
	"strings"
	"testing"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// A weekday outside any promotion: 2026-10-14 is a Wednesday.
func at(hour, minute int) time.Time { return time.Date(2026, 10, 14, hour, minute, 0, 0, peakZone) }

func TestIsPeak(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"start of peak", at(14, 0), true},
		{"inside peak", at(17, 59), true},
		{"end is exclusive", at(18, 0), false},
		{"before peak", at(13, 59), false},
		{"saturday", time.Date(2026, 10, 17, 15, 0, 0, 0, peakZone), false},
		{"promotion makes every hour off-peak", time.Date(2026, 10, 2, 15, 0, 0, 0, peakZone), false},
		{"promotion end is exclusive", time.Date(2026, 10, 8, 15, 0, 0, 0, peakZone), true},
		// The same instant expressed in another zone is still peak.
		{"viewer zone irrelevant", at(15, 0).In(time.FixedZone("CEST", 2*3600)), true},
	}
	for _, c := range cases {
		if got := IsPeak(c.now); got != c.want {
			t.Errorf("%s: IsPeak(%s) = %v, want %v", c.name, c.now, got, c.want)
		}
	}
}

func TestPeakEndsAt(t *testing.T) {
	if got := PeakEndsAt(at(15, 30)); !got.Equal(at(18, 0)) {
		t.Errorf("PeakEndsAt = %s, want 18:00 UTC+8", got)
	}
	if got := PeakEndsAt(at(9, 0)); !got.IsZero() {
		t.Errorf("PeakEndsAt outside peak = %s, want zero", got)
	}
}

func TestBillingModeOf(t *testing.T) {
	credit := client.QuotaLimit{Type: client.QuotaTypeCreditLimit}
	tokens := client.QuotaLimit{Type: client.QuotaTypeTokensLimit}
	mcp := client.QuotaLimit{Type: client.QuotaTypeTimeLimit}
	cases := map[BillingMode][]client.QuotaLimit{
		BillingCredits: {mcp, tokens, credit},
		BillingTokens:  {tokens, mcp},
		BillingUnknown: {mcp},
	}
	for want, limits := range cases {
		if got := BillingModeOf(limits); got != want {
			t.Errorf("BillingModeOf(%v) = %v, want %v", limits, got, want)
		}
	}
}

func TestFormatPeakWarning(t *testing.T) {
	utc := time.UTC
	if got := FormatPeakWarning(at(15, 0), BillingCredits, utc); !strings.Contains(got, "until 10:00 UTC") || !strings.Contains(got, "50%") {
		t.Errorf("credits warning = %q", got)
	}
	if got := FormatPeakWarning(at(15, 0), BillingTokens, utc); !strings.Contains(got, "3×") {
		t.Errorf("tokens warning = %q", got)
	}
	if got := FormatPeakWarning(at(9, 0), BillingCredits, utc); got != "" {
		t.Errorf("off-peak warning = %q, want empty", got)
	}
}
