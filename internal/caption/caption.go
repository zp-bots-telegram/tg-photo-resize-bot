// Package caption renders the reply caption shown under the processed
// photo. Output is HTML; callers should send with parse_mode=HTML.
package caption

import (
	"fmt"
	"html"
	"strings"

	"github.com/zackpollard/tg-photo-resize-bot/internal/exifx"
)

// Render returns the caption for an image with the given original
// dimensions and EXIF tags. Missing EXIF fields are skipped; if no
// fields are present, only the dimensions line is returned.
func Render(origW, origH int, t exifx.Tags) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Original: %d×%d", origW, origH)
	if t.Empty() {
		return b.String()
	}
	b.WriteString("\n\n<pre>")

	type line struct{ label, value string }
	var lines []line

	if cam := joinNonEmpty(" ", t.Make, t.Model); cam != "" {
		lines = append(lines, line{"Camera", cam})
	}
	if t.ISO > 0 {
		lines = append(lines, line{"ISO", fmt.Sprintf("%d", t.ISO)})
	}
	if lens := renderLens(t); lens != "" {
		lines = append(lines, line{"Lens", lens})
	}
	if t.ExposureTime != "" {
		lines = append(lines, line{"Shutter", t.ExposureTime + "s"})
	}

	// Compute padding so values line up.
	width := 0
	for _, l := range lines {
		if n := len(l.label); n > width {
			width = n
		}
	}
	for i, l := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%-*s  %s", width, l.label, html.EscapeString(l.value))
	}
	b.WriteString("</pre>")
	return b.String()
}

func renderLens(t exifx.Tags) string {
	var parts []string
	if t.FocalLength > 0 {
		parts = append(parts, fmt.Sprintf("%dmm", int(t.FocalLength+0.5)))
	}
	if t.FNumber > 0 {
		parts = append(parts, fmt.Sprintf("f/%g", trim2(t.FNumber)))
	}
	return strings.Join(parts, "  ")
}

// trim2 rounds to two decimal places without trailing zeros.
func trim2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
