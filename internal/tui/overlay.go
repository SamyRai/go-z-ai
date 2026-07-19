package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
)

// placeOverlay centers the given content over a backdrop filling width×height.
// The backdrop is a solid field of spaces so the underlying screen body is
// visually obscured (not just painted over with the card). Used by the root
// model to render help/palette/picker overlays above the active screen.
//
// Width/height are clamped to the content's natural size so a too-small
// terminal still shows the card without truncation.
func placeOverlay(width, height int, content string) string {
	if width < 1 || height < 1 {
		return content
	}
	// Build a solid backdrop: `height` lines of `width` spaces, so Place()
	// has something to composite onto and the screen underneath is obscured.
	bg := strings.Repeat(" ", width)
	backdrop := strings.Repeat(bg+"\n", height)
	// Place strips the final newline implicitly via whitespace handling; the
	// trailing \n keeps the rows distinct during JoinVertical elsewhere.
	backdrop = strings.TrimRight(backdrop, "\n")
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// renderOverlayCard wraps content in the bordered modal card style. Thin
// wrapper around uistyle.RenderOverlayCard kept here so the root package's
// help overlay builds through the same shared style as subpackage overlays
// (palette, model picker), with no duplication.
func renderOverlayCard(title, body string) string {
	return uistyle.RenderOverlayCard(title, body)
}
