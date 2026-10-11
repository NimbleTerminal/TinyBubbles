package tinybubbles

import "image/color"

type Style struct{ Foreground, Background color.RGBA }

// Theme is a small palette; undefined style IDs fall back to Normal.
type Theme struct{ Styles [8]Style }

// DefaultTheme uses inverse video for selection on monochrome panels.
// Displayer does not expose color depth, so device configuration supplies it.
func DefaultTheme(monochrome bool) Theme {
	black, white := color.RGBA{A: 255}, color.RGBA{255, 255, 255, 255}
	t := Theme{}
	for i := range t.Styles {
		t.Styles[i] = Style{white, black}
	}
	t.Styles[Selected] = Style{black, white}
	if !monochrome {
		t.Styles[Accent].Foreground = color.RGBA{64, 220, 255, 255}
		t.Styles[Muted].Foreground = color.RGBA{155, 165, 180, 255}
		t.Styles[Selected] = Style{black, color.RGBA{64, 220, 255, 255}}
	}
	return t
}

func (t Theme) style(id StyleID) Style {
	if int(id) >= len(t.Styles) {
		id = Normal
	}
	return t.Styles[id]
}
