package tui

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image/color"
	"io"
	"strings"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// cells is a little canvas with one of everything a picture must handle.
func cells() *canvas.Canvas {
	c := canvas.New(12, 3)
	for i, r := range "&<a>" { // characters XML would choke on, in a run
		c.Set(i, 0, r, canvas.InkLeaf)
	}
	c.Set(5, 0, 'x', canvas.InkNone)
	c.Set(6, 0, '~', canvas.InkCloud) // faint
	c.Set(7, 0, '%', canvas.InkLeaf)
	c.Shade(7, 0) // out of the light
	for i, r := range []rune("█▀▄░") {
		c.Set(i, 1, r, canvas.InkMoon)
	}
	c.Set(0, 2, '✿', canvas.InkBlossom)
	c.Set(1, 2, '✿', canvas.InkBlossom)
	c.Set(3, 2, '✿', canvas.InkBlossom) // a gap splits the run
	return c
}

func svg(t *testing.T, c *canvas.Canvas, th Theme, caption string) string {
	t.Helper()
	var b bytes.Buffer
	if err := SVG(&b, c, th, Frappe, caption); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// wellFormed fails the test unless s parses as XML from start to end.
func wellFormed(t *testing.T, s string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := d.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("not well-formed XML: %v\n%s", err, s)
		}
	}
}

func TestSVGGolden(t *testing.T) {
	out := svg(t, cells(), DefaultTheme(), "txoko · test")
	wellFormed(t, out)
	assertGolden(t, "snap_cells", out)
}

func TestSVGCells(t *testing.T) {
	out := svg(t, cells(), DefaultTheme(), "")
	for _, want := range []string{
		`>&amp;&lt;a&gt;</text>`,                     // escaped, and one run
		`x="28.0 38.0 48.0 58.0"`,                    // each character in its own cell
		`fill="#a6d189" font-weight="bold"`,          // a leaf: bright green, bold
		`opacity="0.55">~`,                           // a faint cloud
		`opacity="0.55">%`,                           // a shaded leaf is dimmed
		`fill="#c6d0f5">x`,                           // no ink: the foreground color
		`<rect x="28" y="48" width="10" height="20"`, // █ fills its cell
		`<rect x="38" y="48" width="10" height="10"`, // ▀ the top half
		`<rect x="48" y="58" width="10" height="10"`, // ▄ the bottom half
		`fill-opacity="0.3"`,                         // ░ a faint texture
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	if strings.Count(out, "✿") != 3 || strings.Count(out, ">✿✿</text>") != 1 {
		t.Error("the gap should split the flowers into two runs")
	}
	if strings.Contains(out, "txoko ·") {
		t.Error("no caption was asked for")
	}
}

func TestSVGPlainTheme(t *testing.T) {
	out := svg(t, cells(), PlainTheme(), "")
	if strings.Contains(out, "#a6d189") || strings.Contains(out, "font-weight") {
		t.Errorf("a plain theme draws everything in the foreground color:\n%s", out)
	}
}

func TestSVGWriteError(t *testing.T) {
	if err := SVG(failingWriter{}, cells(), DefaultTheme(), Frappe, ""); err == nil {
		t.Error("a write error is reported")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestANSIIndex(t *testing.T) {
	if _, ok := ansiIndex(color.RGBA{R: 1}); ok {
		t.Error("a true color isn't one of the 16")
	}
	if _, ok := ansiIndex(nil); ok {
		t.Error("no color isn't one of the 16")
	}
}

// TestSnap: every scene, by day and by night, makes a well-formed picture
// of the default size, captioned with what it shows.
func TestSnap(t *testing.T) {
	for _, name := range SceneNames() {
		for _, sky := range []string{"day", "night"} {
			var b bytes.Buffer
			o := OnceOptions{Scene: name, Sky: sky, Season: "spring", Growth: 1, Seed: 42, Now: noon, MoonPhase: &testMoon}
			if err := Snap(&b, o, Frappe); err != nil {
				t.Fatal(err)
			}
			out := b.String()
			wellFormed(t, out)
			if !strings.Contains(out, `viewBox="0 0 656 526"`) {
				t.Errorf("%s %s: not the default %dx%d size", name, sky, SnapWidth, SnapHeight)
			}
			if want := "txoko · " + name + " · spring · " + sky; !strings.Contains(out, want) {
				t.Errorf("caption %q missing", want)
			}
		}
	}
	var b bytes.Buffer
	if err := Snap(&b, OnceOptions{Width: 30, Height: 10, Growth: 1, Now: noon}, Frappe); err != nil || !strings.Contains(b.String(), `viewBox="0 0 356 286"`) {
		t.Errorf("a chosen size: %v", err)
	}
}
