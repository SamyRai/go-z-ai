package models

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/modelview"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// This file holds the pure renderers for the Models tab's right-hand preview
// pane and full-screen detail view, plus the small formatting helpers they
// share. Splitting them out mirrors usage/heatmap.go: the state-machine and
// message routing live in model.go, the visual presentation lives here.

// capBadges renders capability codes as colored badge strings for the detail
// / preview panes, where there's room for a touch more flair than the table.
func capBadges(caps []string) string {
	names := modelview.CapabilityNames(caps)
	if len(names) == 0 {
		return uistyle.Subtle.Render("no capabilities listed")
	}
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(uistyle.ColorAccentBg).
		Padding(0, 1)
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = badgeStyle.Render(n)
	}
	return strings.Join(out, " ")
}

// displayName is the model's Name (live or catalog), falling back to the raw
// ID so the headline is always meaningful even for uncataloged models.
func displayName(md client.ModelDetails) string {
	if md.Name != "" {
		return md.Name
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
		uistyle.Subtle.Render("Context"), modelview.FormatTokens(md.ContextSize),
		uistyle.Subtle.Render("max out:"), modelview.FormatTokens(md.MaxOutput))
	fmt.Fprintf(&b, "%-11s %s   %s %s\n\n",
		uistyle.Subtle.Render("Released"), modelview.ReleaseDate(md),
		uistyle.Subtle.Render("by"), md.OwnedBy)

	// Pricing block.
	b.WriteString(uistyle.SectionTitle.Render("Pricing / 1M tokens"))
	b.WriteString("\n")
	if md.Pricing != nil {
		fmt.Fprintf(&b, "  %-8s %s   %-8s %s\n",
			"input", modelview.FormatRate(md.Pricing.Input),
			"output", modelview.FormatRate(md.Pricing.Output))
		if md.Pricing.Cached > 0 {
			fmt.Fprintf(&b, "  %-8s %s\n",
				"cached", modelview.FormatRate(md.Pricing.Cached))
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
	if desc := md.Description; desc != "" {
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
	fmt.Fprintf(&b, "%-13s %s\n", uistyle.Subtle.Render("Released"), modelview.ReleaseDate(md))
	b.WriteString("\n")

	// Limits block.
	b.WriteString(uistyle.SectionTitle.Render("Limits"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %-13s %s tokens\n", "Context", modelview.FormatTokens(md.ContextSize))
	fmt.Fprintf(&b, "  %-13s %s tokens\n", "Max output", modelview.FormatTokens(md.MaxOutput))
	b.WriteString("\n")

	// Pricing block — fuller than the preview.
	b.WriteString(uistyle.SectionTitle.Render("Pricing (per 1M tokens, USD)"))
	b.WriteString("\n")
	if md.Pricing != nil {
		fmt.Fprintf(&b, "  %-13s %s\n", "Input", modelview.FormatRate(md.Pricing.Input))
		fmt.Fprintf(&b, "  %-13s %s\n", "Output", modelview.FormatRate(md.Pricing.Output))
		if md.Pricing.Cached > 0 {
			fmt.Fprintf(&b, "  %-13s %s\n", "Cached input", modelview.FormatRate(md.Pricing.Cached))
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
	if desc := md.Description; desc != "" {
		b.WriteString(uistyle.SectionTitle.Render("About"))
		b.WriteString("\n")
		b.WriteString(wrap(desc, w))
		b.WriteString("\n")
	}
	return b.String()
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
