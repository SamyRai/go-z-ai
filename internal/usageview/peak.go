package usageview

import (
	"fmt"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// GLM Coding Plan peak hours are Monday–Friday 14:00–18:00 UTC+8 for every
// user worldwide — not shifted to the viewer's zone, and independent of any
// monitor-timezone override. The quota API applies peak pricing server-side
// without exposing it, so it is mirrored here
// (docs.z.ai/devpack/overview, checked 2026-10-02). What peak costs depends
// on the plan's billing mode (see BillingMode).
var peakZone = time.FixedZone("UTC+8", 8*3600)

const (
	peakStartHour = 14
	peakEndHour   = 18
)

// offPeakPromotions are announced periods, [from, to), during which every hour
// is billed at the off-peak rate.
var offPeakPromotions = []struct{ from, to time.Time }{
	// "Double Festival", 2026-09-25 to 2026-10-07 inclusive.
	{time.Date(2026, 9, 25, 0, 0, 0, 0, peakZone), time.Date(2026, 10, 8, 0, 0, 0, 0, peakZone)},
}

// BillingMode is how a coding plan meters usage, which decides what peak
// hours cost.
type BillingMode int

const (
	BillingUnknown BillingMode = iota
	// BillingCredits: credit-based plans (sold since 2026-07-30) bill peak
	// usage at the standard credit rate and off-peak usage at 50%.
	BillingCredits
	// BillingTokens: legacy token plans count GLM-5.3 at 3× during peak (1×
	// off-peak) and GLM-5.3-Flash at 1.2× (0.4×).
	BillingTokens
)

// BillingModeOf infers the billing mode from a plan's quota windows.
func BillingModeOf(limits []client.QuotaLimit) BillingMode {
	mode := BillingUnknown
	for _, l := range limits {
		switch l.Type {
		case client.QuotaTypeCreditLimit:
			return BillingCredits
		case client.QuotaTypeTokensLimit:
			mode = BillingTokens
		}
	}
	return mode
}

// IsPeak reports whether now falls in the weekday peak window, outside any
// off-peak promotion.
func IsPeak(now time.Time) bool {
	for _, p := range offPeakPromotions {
		if !now.Before(p.from) && now.Before(p.to) {
			return false
		}
	}
	t := now.In(peakZone)
	if d := t.Weekday(); d == time.Saturday || d == time.Sunday {
		return false
	}
	return t.Hour() >= peakStartHour && t.Hour() < peakEndHour
}

// PeakEndsAt returns when the current peak window ends, or the zero time if
// now is not in peak.
func PeakEndsAt(now time.Time) time.Time {
	if !IsPeak(now) {
		return time.Time{}
	}
	t := now.In(peakZone)
	return time.Date(t.Year(), t.Month(), t.Day(), peakEndHour, 0, 0, 0, peakZone)
}

// FormatPeakWarning returns a one-line notice while peak hours are on, ""
// otherwise. The end time is shown in loc (the viewer's zone). Plain text;
// the caller styles it.
func FormatPeakWarning(now time.Time, mode BillingMode, loc *time.Location) string {
	end := PeakEndsAt(now)
	if end.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	until := end.In(loc).Format("15:04 MST")
	switch mode {
	case BillingCredits:
		return fmt.Sprintf("⚡ peak hours until %s: credits bill at the full rate (off-peak costs 50%%)", until)
	case BillingTokens:
		return fmt.Sprintf("⚡ peak hours until %s: GLM-5.3 counts 3× (GLM-5.3-Flash 1.2×)", until)
	default:
		return fmt.Sprintf("⚡ peak hours until %s: plan usage is billed at the peak rate", until)
	}
}
