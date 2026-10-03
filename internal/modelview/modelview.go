// Package modelview holds the presentation helpers for model metadata —
// context sizes, prices, capabilities — shared by the CLI's models commands
// and the TUI's Models tab and model picker, so their output can't drift.
package modelview

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Unknown is rendered for a value the catalog doesn't know.
const Unknown = "—"

// FormatTokens renders a token count compactly in binary units, as model
// limits are published: 1048576 → "1M", 131072 → "128K", 0 → Unknown.
func FormatTokens(n int) string {
	switch {
	case n <= 0:
		return Unknown
	case n >= 1<<20:
		return trimFloat(float64(n)/(1<<20), 1) + "M"
	case n >= 1<<10:
		return strconv.Itoa(n>>10) + "K"
	}
	return strconv.Itoa(n)
}

// FormatRate renders a USD-per-1M-token rate with two decimals, or three
// for sub-cent precision: "$1.40", "$0.075", "$0.04", "$0".
func FormatRate(v float64) string {
	switch {
	case v == 0:
		return "$0"
	case v >= 0.1:
		return fmt.Sprintf("$%.2f", v)
	}
	return "$" + strings.TrimSuffix(fmt.Sprintf("%.3f", v), "0")
}

// Rates returns the input, output, and cached-input columns for a model's
// pricing, with Unknown for anything not known.
func Rates(p *client.Pricing) (in, out, cached string) {
	if p == nil {
		return Unknown, Unknown, Unknown
	}
	cached = Unknown
	if p.Cached > 0 {
		cached = FormatRate(p.Cached)
	}
	return FormatRate(p.Input), FormatRate(p.Output), cached
}

// capabilities is the display order and compact code of each capability.
var capabilities = []struct{ cap, code string }{
	{client.CapText, "T"},
	{client.CapVision, "V"},
	{client.CapVideo, "vid"},
	{client.CapFile, "f"},
	{client.CapThinking, "th"},
	{client.CapTools, "tl"},
	{client.CapCode, "c"},
	{client.CapOCR, "ocr"},
	{client.CapAudio, "asr"},
}

// Capabilities joins capability names in display order ("text,vision,…"),
// or Unknown for none.
func Capabilities(caps []string) string { return render(caps, false) }

// CapabilityNames returns the capability names in display order.
func CapabilityNames(caps []string) []string {
	var out []string
	for _, c := range capabilities {
		if slices.Contains(caps, c.cap) {
			out = append(out, c.cap)
		}
	}
	return out
}

// CapabilityCodes joins compact capability codes ("T V th tl") for narrow
// table columns, or Unknown for none.
func CapabilityCodes(caps []string) string { return render(caps, true) }

func render(caps []string, codes bool) string {
	var out []string
	for _, c := range capabilities {
		if !slices.Contains(caps, c.cap) {
			continue
		}
		if codes {
			out = append(out, c.code)
		} else {
			out = append(out, c.cap)
		}
	}
	if len(out) == 0 {
		return Unknown
	}
	if codes {
		return strings.Join(out, " ")
	}
	return strings.Join(out, ",")
}

// trimFloat formats v with up to prec decimals, dropping trailing zeros.
func trimFloat(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// ReleaseDate renders a model's release date, or Unknown.
func ReleaseDate(m client.ModelDetails) string {
	if t := m.CreatedTime(); !t.IsZero() {
		return t.UTC().Format("2006-01-02")
	}
	return Unknown
}

// Summary is a one-line description of a model's limits and price for
// pickers: "1M ctx · $1.40/$4.40".
func Summary(m client.ModelDetails) string {
	in, out, _ := Rates(m.Pricing)
	if m.IsFree() {
		return fmt.Sprintf("%s ctx · free", FormatTokens(m.ContextSize))
	}
	return fmt.Sprintf("%s ctx · %s/%s", FormatTokens(m.ContextSize), in, out)
}
