package models

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// This file holds the pure renderers for the Models tab's right-hand preview
// pane and full-screen detail view, plus the small formatting helpers they
// share. Splitting them out mirrors usage/heatmap.go: the state-machine and
// message routing live in model.go, the visual presentation lives here.

// formatContext renders a token count as a compact human string ("200K",
// "128K", "—" for zero/unknown).
func formatContext(n int) string {
	if n <= 0 {
		return "—"
	}
	if n >= 1000 {
		// Round to nearest K, no decimals for values >= 10K, one decimal below.
		k := float64(n) / 1000
		if k >= 100 {
			return fmt.Sprintf("%dK", int(k))
		}
		return fmt.Sprintf("%.0fK", k)
	}
	return fmt.Sprintf("%d", n)
}

// formatPrice renders a per-1M-token USD rate, or "—" when unknown. Always
// two decimals so the column lines up.
func formatPrice(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("$%.2f", v)
}

// formatCaps renders a model's capability list as a row of compact pill-ish
// codes suitable for the table's narrow CAPS column: T (text), V (vision),
// th (thinking), tl (tools), c (code), ocr. Capabilities with no compact
// code are skipped. Order is fixed for readability.
func formatCaps(caps []string) string {
	if len(caps) == 0 {
		return "—"
	}
	// Fixed display order.
	order := []struct {
		cap, code string
	}{
		{client.CapText, "T"},
		{client.CapVision, "V"},
		{client.CapThinking, "th"},
		{client.CapTools, "tl"},
		{client.CapCode, "c"},
		{client.CapOCR, "ocr"},
	}
	var out []string
	for _, o := range order {
		for _, c := range caps {
			if c == o.cap {
				out = append(out, o.code)
				break
			}
		}
	}
	if len(out) == 0 {
		return "—"
	}
	return strings.Join(out, " ")
}

// capBadges renders capability codes as colored badge strings for the detail
// / preview panes, where there's room for a touch more flair than the table.
func capBadges(caps []string) string {
	if len(caps) == 0 {
		return uistyle.Subtle.Render("no capabilities listed")
	}
	labels := map[string]string{
		client.CapText:     "text",
		client.CapVision:   "vision",
		client.CapThinking: "thinking",
		client.CapTools:    "tools",
		client.CapCode:     "code",
		client.CapOCR:      "ocr",
	}
	order := []string{
		client.CapText, client.CapVision, client.CapThinking,
		client.CapTools, client.CapCode, client.CapOCR,
	}
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(uistyle.ColorAccentBg).
		Padding(0, 1)
	var out []string
	for _, cap := range order {
		for _, c := range caps {
			if c == cap {
				out = append(out, badgeStyle.Render(labels[cap]))
				break
			}
		}
	}
	return strings.Join(out, " ")
}

// formatReleased renders the model's created epoch as a "Mon YYYY" string, or
// "—" when unknown (zero).
func formatReleased(epoch int64) string {
	if epoch <= 0 {
		return "—"
	}
	return time.Unix(epoch, 0).UTC().Format("Jan 2006")
}

// displayName prefers a live-API Name, falls back to the catalog name, then to
// the raw ID — so the headline is always meaningful even for uncataloged models.
func displayName(md client.ModelDetails) string {
	if md.Name != "" {
		return md.Name
	}
	if md.CatalogName != "" {
		return md.CatalogName
	}
	return md.ID
}

// tierLabel renders the tier as a small uppercase tag, or empty string.
func tierLabel(md client.ModelDetails) string {
	if md.Tier == "" {
		return ""
	}
	return uistyle.Subtle.Render(strings.ToUpper(md.Tier))
}

// renderPreview renders the right-hand detail card shown alongside the table
// in wide terminals. width is the available width for the pane (caller has
// already subtracted any separator). It must not render a border — the root
// model already wraps the whole screen in uistyle.Panel, and nesting a second
// Panel would double-box.
func renderPreview(md client.ModelDetails, width int) string {
	w := width
	if w < 24 {
		w = 24 // floor so the layout doesn't collapse on marginal widths
	}

	head := displayName(md)
	if tl := tierLabel(md); tl != "" {
		head += "  " + tl
	}

	var b strings.Builder
	b.WriteString(uistyle.SectionTitle.Render(head))
	b.WriteString("\n")

	// Specs row.
	fmt.Fprintf(&b, "%-11s %s\n",
		uistyle.Subtle.Render("ID"), md.ID)
	if md.Family != "" {
		fmt.Fprintf(&b, "%-11s %s\n",
			uistyle.Subtle.Render("Family"), md.Family)
	}
	fmt.Fprintf(&b, "%-11s %s   %s %s\n",
		uistyle.Subtle.Render("Context"), formatContext(md.ContextSize),
		uistyle.Subtle.Render("max out:"), formatContext(md.MaxOutput))
	fmt.Fprintf(&b, "%-11s %s   %s %s\n\n",
		uistyle.Subtle.Render("Released"), formatReleased(md.Created),
		uistyle.Subtle.Render("by"), md.OwnedBy)

	// Pricing block.
	b.WriteString(uistyle.SectionTitle.Render("Pricing / 1M tokens"))
	b.WriteString("\n")
	if md.Pricing != nil {
		fmt.Fprintf(&b, "  %-8s %s   %-8s %s\n",
			"input", formatPrice(md.Pricing.Input),
			"output", formatPrice(md.Pricing.Output))
		if md.Pricing.Cached > 0 {
			fmt.Fprintf(&b, "  %-8s %s\n",
				"cached", formatPrice(md.Pricing.Cached))
		}
	} else {
		b.WriteString(uistyle.Subtle.Render("  — pricing not available"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Capabilities.
	b.WriteString(uistyle.SectionTitle.Render("Capabilities"))
	b.WriteString("\n")
	b.WriteString(capBadges(md.Capabilities))
	b.WriteString("\n\n")

	// Description (word-wrapped to pane width).
	if desc := descriptionText(md); desc != "" {
		b.WriteString(uistyle.Subtle.Render(wrap(desc, w)))
		b.WriteString("\n")
	}
	return b.String()
}

// renderDetail renders the full-screen Enter view, intended for a viewport.
// width is the full available width. Richer than the preview — shows
// everything we know, including cache-storage pricing when present.
func renderDetail(md client.ModelDetails, width int) string {
	w := width
	if w < 40 {
		w = 40
	}

	head := displayName(md)
	if tl := tierLabel(md); tl != "" {
		head += "  " + tl
	}

	var b strings.Builder
	b.WriteString(uistyle.SectionTitle.Render(head))
	b.WriteString("\n\n")

	// Identity block.
	fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("ID"), md.ID)
	if md.Family != "" {
		fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("Family"), md.Family)
	}
	if md.Tier != "" {
		fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("Tier"), md.Tier)
	}
	fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("Owned by"), md.OwnedBy)
	fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("Released"), formatReleased(md.Created))
	b.WriteString("\n")

	// Limits block.
	b.WriteString(uistyle.SectionTitle.Render("Limits"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %-13s %s tokens\n", "Context", formatContext(md.ContextSize))
	fmt.Fprintf(&b, "  %-13s %s tokens\n", "Max output", formatContext(md.MaxOutput))
	b.WriteString("\n")

	// Pricing block — fuller than the preview.
	b.WriteString(uistyle.SectionTitle.Render("Pricing (per 1M tokens, USD)"))
	b.WriteString("\n")
	if md.Pricing != nil {
		fmt.Fprintf(&b, "  %-13s %s\n", "Input", formatPrice(md.Pricing.Input))
		fmt.Fprintf(&b, "  %-13s %s\n", "Output", formatPrice(md.Pricing.Output))
		if md.Pricing.Cached > 0 {
			fmt.Fprintf(&b, "  %-13s %s\n", "Cached input", formatPrice(md.Pricing.Cached))
		}
		if md.Pricing.CacheStore > 0 {
			fmt.Fprintf(&b, "  %-13s %s\n", "Cache storage", formatPrice(md.Pricing.CacheStore))
		}
		if md.IsFree() {
			b.WriteString("  " + uistyle.ToastInfo.Render("free model") + "\n")
		}
	} else {
		b.WriteString(uistyle.Subtle.Render("  — pricing not available for this model"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Capabilities.
	b.WriteString(uistyle.SectionTitle.Render("Capabilities"))
	b.WriteString("\n")
	b.WriteString("  " + capBadges(md.Capabilities))
	b.WriteString("\n\n")

	// Description.
	if desc := descriptionText(md); desc != "" {
		b.WriteString(uistyle.SectionTitle.Render("About"))
		b.WriteString("\n")
		b.WriteString(wrap(desc, w))
		b.WriteString("\n")
	}
	return b.String()
}

// descriptionText returns the live Description if present, else the catalog
// description. Empty if neither.
func descriptionText(md client.ModelDetails) string {
	if md.Description != "" {
		return md.Description
	}
	return md.CatalogDescription
}

// wrap word-wraps s to width, preserving existing newlines. Cheap
// implementation (no dependency on a wrapping lib) — good enough for short
// one-paragraph model blurbs.
func wrap(s string, width int) string {
	if width < 10 {
		return s
	}
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			out.WriteString("\n")
			continue
		}
		col := 0
		for i, w := range words {
			wlen := len(w)
			if i == 0 {
				out.WriteString(w)
				col = wlen
				continue
			}
			if col+1+wlen > width {
				out.WriteString("\n")
				out.WriteString(w)
				col = wlen
			} else {
				out.WriteString(" ")
				out.WriteString(w)
				col += 1 + wlen
			}
		}
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}
