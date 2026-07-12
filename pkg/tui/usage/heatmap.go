package usage

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/pkg/usageview"
)

// heatmapRampColors maps each usageview.HeatmapLevels index to a color of
// increasing intensity. Colors are built via lipgloss.Color only, never raw
// ANSI, so Bubble Tea's colorprofile layer downsamples them automatically
// for NO_COLOR/16-color/dumb terminals; the plain density characters
// (" ░▒▓█") remain the terminal fallback in that case, so no separate
// no-color code path is needed here.
var heatmapRampColors = []color.Color{
	lipgloss.Color("0"),
	lipgloss.Color("22"),
	lipgloss.Color("28"),
	lipgloss.Color("34"),
	lipgloss.Color("46"),
}

// renderHeatmapRow colorizes one heatmapview.HeatmapBlocks row character by
// character according to each value's own scaled intensity.
func renderHeatmapRow(values []int64) string {
	var max int64
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	blocks := usageview.HeatmapBlocks(values)
	runes := []rune(blocks)
	out := ""
	for i, r := range runes {
		level := usageview.HeatmapLevelIndex(values[i], max)
		style := lipgloss.NewStyle().Foreground(heatmapRampColors[level])
		out += style.Render(string(r))
	}
	return out
}
