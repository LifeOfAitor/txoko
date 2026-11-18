package tui

import (
	"fmt"
	"html"
	"image/color"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// Palette gives real colors to the terminal's 16 ANSI colors, for pictures
// that live outside a terminal.
type Palette struct {
	Name       string
	Background string
	Foreground string
	ANSI       [16]string // black, red, green, yellow, blue, magenta, cyan, white, then the bright ones
}

// Frappe is Catppuccin Frappé, the palette of txoko's screenshots.
var Frappe = Palette{
	Name:       "frappe",
	Background: "#303446",
	Foreground: "#c6d0f5",
	ANSI: [16]string{
		"#51576d", "#e78284", "#a6d189", "#e5c890", "#8caaee", "#f4b8e4", "#81c8be", "#b5bfe2",
		"#626880", "#e78284", "#a6d189", "#e5c890", "#8caaee", "#f4b8e4", "#81c8be", "#a5adce",
	},
}

// Default size of a snapshot, in cells: room for a scene to breathe.
const (
	SnapWidth  = 60
	SnapHeight = 22
)

// Snap draws the scene o asks for once, like Once but without the info
// panel, and writes it to w as an SVG picture in palette p, with a caption
// that says what it shows.
func Snap(w io.Writer, o OnceOptions, p Palette) error {
	if o.Width <= 0 || o.Height <= 0 {
		o.Width, o.Height = SnapWidth, SnapHeight
	}
	f := still(o)
	sky := "day"
	if f.night {
		sky = "night"
	}
	return SVG(w, f.canvas, DefaultTheme(), p, "txoko · "+f.scene+" · "+f.season.String()+" · "+sky)
}

// Sizes of a cell in the picture, in pixels.
const (
	cellW   = 10.0
	cellH   = 20.0
	fontPx  = 16.0
	padding = 28.0 // a whole number of pixels, so blocks line up
)

// SVG draws a canvas as an SVG picture: every cell in its exact place,
// colored as the theme would color it in a terminal using palette p. The
// moon's half blocks become real rectangles, so it stays round whatever
// font the viewer has. caption, if not empty, is written below.
func SVG(w io.Writer, c *canvas.Canvas, t Theme, p Palette, caption string) error {
	width := float64(c.Width())*cellW + 2*padding
	height := float64(c.Height())*cellH + 2*padding
	if caption != "" {
		height += cellH * 1.5
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect width="100%%" height="100%%" rx="14" fill="%s"/>`+"\n", p.Background)
	fmt.Fprintf(&b, `<g font-family="'JetBrains Mono','SF Mono',Menlo,'DejaVu Sans Mono',monospace" font-size="%.0f">`+"\n", fontPx)

	for y := range c.Height() {
		row := c.Row(y)
		for i := 0; i < len(row); {
			cell := row[i]
			if cell.Rune == ' ' {
				i++
				continue
			}
			fill, weight, opacity := cellStyle(t, p, cell)
			if block(&b, cell.Rune, i, y, fill, opacity) {
				i++
				continue
			}
			// A run of cells with the same look, each at its own x.
			var text strings.Builder
			var xs []string
			for ; i < len(row) && row[i].Rune != ' ' && !isBlock(row[i].Rune) && row[i].Ink == cell.Ink && row[i].Shadow == cell.Shadow; i++ {
				text.WriteRune(row[i].Rune)
				xs = append(xs, fmt.Sprintf("%.1f", padding+float64(i)*cellW))
			}
			fmt.Fprintf(&b, `<text x="%s" y="%.1f" fill="%s"%s%s>%s</text>`+"\n",
				strings.Join(xs, " "), padding+float64(y)*cellH+fontPx*0.95, fill, weight, opacity, html.EscapeString(text.String()))
		}
	}
	if caption != "" {
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="%s" font-size="13" opacity="0.8">%s</text>`+"\n",
			padding, height-padding*0.75, p.ANSI[8], html.EscapeString(caption))
	}
	b.WriteString("</g>\n</svg>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// cellStyle is the fill color and the font-weight and opacity attributes
// of a cell.
func cellStyle(t Theme, p Palette, cell canvas.Cell) (fill, weight, opacity string) {
	fill = p.Foreground
	if cell.Ink == canvas.InkNone || t.plain {
		return fill, "", ""
	}
	style := t.Ink[cell.Ink]
	if c, ok := ansiIndex(style.GetForeground()); ok {
		fill = p.ANSI[c]
	}
	bold, faint := style.GetBold(), style.GetFaint()
	if cell.Shadow {
		bold, faint = false, true
	}
	if bold {
		weight = ` font-weight="bold"`
	}
	if faint {
		opacity = ` opacity="0.55"`
	}
	return fill, weight, opacity
}

// ansiIndex is which of the 16 ANSI colors c is.
func ansiIndex(c color.Color) (int, bool) {
	if b, ok := c.(ansi.BasicColor); ok && b < 16 {
		return int(b), true
	}
	return 0, false
}

func isBlock(r rune) bool { return strings.ContainsRune("█▀▄░", r) }

// block draws the moon's half blocks as rectangles; it reports false for
// any other rune.
func block(b *strings.Builder, r rune, x, y int, fill, opacity string) bool {
	x0, y0 := padding+float64(x)*cellW, padding+float64(y)*cellH
	rect := func(top, h float64, extra string) {
		fmt.Fprintf(b, `<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="%s" shape-rendering="crispEdges"%s%s/>`+"\n",
			x0, y0+top, cellW, h, fill, opacity, extra)
	}
	switch r {
	case '█':
		rect(0, cellH, "")
	case '▀':
		rect(0, cellH/2, "")
	case '▄':
		rect(cellH/2, cellH/2, "")
	case '░':
		rect(0, cellH, ` fill-opacity="0.3"`)
	default:
		return false
	}
	return true
}
