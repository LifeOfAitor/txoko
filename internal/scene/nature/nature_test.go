package nature

import (
	"strings"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

func TestCacheFollowsSeedAndSize(t *testing.T) {
	a := Grow(1, 40, 12)
	b := Grow(2, 40, 12)
	c := Grow(1, 41, 12)
	if len(a.Marks) == 0 || equalMarks(a, b) || equalMarks(a, c) {
		t.Error("cache returned a tree for the wrong seed or size")
	}
	if !equalMarks(Grow(1, 40, 12), a) {
		t.Error("cache should return the same tree for the same inputs")
	}
}

func equalMarks(a, b Tree) bool {
	if len(a.Marks) != len(b.Marks) {
		return false
	}
	for i := range a.Marks {
		if a.Marks[i] != b.Marks[i] {
			return false
		}
	}
	return true
}

func TestGlyphs(t *testing.T) {
	g := &grower{}
	cases := []struct {
		dx, dy int
		want   rune
	}{
		{0, -1, '|'},
		{-1, -1, '\\'},
		{1, -1, '/'},
		{1, 1, '\\'},
		{-1, 1, '/'},
		{0, 1, '|'},
	}
	for _, c := range cases {
		if got := g.glyph(limb, c.dx, c.dy); got != c.want {
			t.Errorf("glyph(%d,%d) = %q, want %q", c.dx, c.dy, got, c.want)
		}
	}
}

func TestTriangle(t *testing.T) {
	want := []int{2, 1, 0, 1, 2, 3, 2, 1, 0, 1}
	for i, w := range want {
		if got := Triangle(i-2, 4); got != w {
			t.Errorf("Triangle(%d, 4) = %d, want %d", i-2, got, w)
		}
	}
	if Triangle(3, 1) != 0 {
		t.Error("triangle with n<=1 should be 0")
	}
}

func TestNoiseIsStableAndUniform(t *testing.T) {
	if a, b := Noise(3, 4, 1), Noise(3, 4, 1); a != b {
		t.Fatal("noise must be deterministic")
	}
	below := 0
	for i := range 10000 {
		v := Noise(i%100, i/100, 5)
		if v < 0 || v >= 1 {
			t.Fatalf("noise out of range: %v", v)
		}
		if v < 0.5 {
			below++
		}
	}
	if below < 4700 || below > 5300 {
		t.Errorf("noise looks biased: %d of 10000 below 0.5", below)
	}
}

func TestBigStarsSparkle(t *testing.T) {
	c := canvas.New(9, 9)
	for f := range 60 {
		drawStar(c, 4, 4, big, 0, f)
	}
	rays := 0
	for f := range 30 {
		c.Clear()
		drawStar(c, 4, 4, big, 0, f)
		if c.At(3, 4).Rune == '·' && c.At(5, 4).Rune == '·' && c.At(4, 3).Rune == '·' && c.At(4, 5).Rune == '·' {
			rays++
		}
	}
	if rays == 0 || rays == 30 {
		t.Errorf("a big star's rays should pulse, shown in %d of 30 frames", rays)
	}
}

func TestBoxContains(t *testing.T) {
	b := box{2, 2, 4, 3}
	if !b.contains(2, 2, 0) || b.contains(1, 2, 0) || !b.contains(1, 2, 1) {
		t.Error("box.contains is wrong")
	}
}

// flat is a canopy with nothing but ground below row floor and, if set, a
// spot where the moon should hang.
type flat struct {
	floor         int
	cx, top, maxR int
}

func (f flat) UnderCanopy(_, y, margin int) bool {
	return y >= f.floor-margin
}
func (f flat) Floor() int { return f.floor }

type framed struct{ flat }

func (f framed) MoonFrame() (int, int, int) { return f.cx, f.top, f.maxR }

func moonCells(c *canvas.Canvas) (minX, maxX, minY int, n int) {
	minX, minY = c.Width(), c.Height()
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			if cell.Ink == canvas.InkMoon || cell.Ink == canvas.InkMoonDark {
				minX, maxX, minY, n = min(minX, x), max(maxX, x), min(minY, y), n+1
			}
		}
	}
	return minX, maxX, minY, n
}

// TestMoonFramer: a scene can hang the moon where it likes; with no room
// it goes back to the top right corner.
func TestMoonFramer(t *testing.T) {
	c := canvas.New(60, 20)
	DrawSky(c, 1, framed{flat{floor: 18, cx: 20, top: 6, maxR: 3}}, 0.5, 0)
	if minX, maxX, minY, n := moonCells(c); n == 0 || minX < 17 || maxX > 22 || minY != 6 {
		t.Errorf("framed moon at x %d..%d, y %d (%d cells)", minX, maxX, minY, n)
	}

	c = canvas.New(60, 20)
	DrawSky(c, 1, framed{flat{floor: 18, maxR: 1}}, 0.5, 0)
	if minX, _, _, n := moonCells(c); n == 0 || minX < 40 {
		t.Errorf("with no room the moon belongs in the corner, found it from x %d", minX)
	}
}

// TestCloudsPassBehind: clouds drift across their rows, behind whatever
// is already drawn.
func TestCloudsPassBehind(t *testing.T) {
	seen := 0
	for f := 0; f < 600; f += 37 {
		c := canvas.New(60, 20)
		for x := 30; x < 60; x++ {
			c.Set(x, 5, '#', canvas.InkRock) // something in the way
		}
		DrawClouds(c, 1, []int{1, 5, 19}, f)
		for x := 30; x < 60; x++ {
			if c.At(x, 5).Rune != '#' {
				t.Fatalf("frame %d: a cloud covered (%d,5)", f, x)
			}
		}
		seen += strings.Count(c.String(), "~")
	}
	if seen == 0 {
		t.Error("clouds should drift by")
	}
}
