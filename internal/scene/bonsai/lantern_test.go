package bonsai

import (
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

func night(g float64, frame int, season scene.Season) *canvas.Canvas {
	c := canvas.New(60, 20)
	Draw(c, testSeed, scene.State{Growth: g, Frame: frame, Night: true, Season: season})
	return c
}

func find(c *canvas.Canvas, ink canvas.Ink) (x, y int, ok bool) {
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			if cell.Ink == ink {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func TestNightHasALanternAndShadows(t *testing.T) {
	for _, season := range scene.Seasons {
		c := night(1, 10, season)
		n := inks(c)
		if n[canvas.InkLantern] != 1 {
			t.Errorf("%s night: %d lanterns, want 1", season, n[canvas.InkLantern])
		}
		if n[canvas.InkGlow] == 0 {
			t.Errorf("%s night: nothing lit by the lantern's glow", season)
		}
		shadows := 0
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				if cell.Shadow {
					shadows++
				}
			}
		}
		if shadows == 0 {
			t.Errorf("%s night: nothing in shadow", season)
		}
	}
}

func TestDayHasNoLanternOrShadows(t *testing.T) {
	c := canvas.New(60, 20)
	Draw(c, testSeed, scene.State{Growth: 1})
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			if cell.Shadow || cell.Ink == canvas.InkLantern || cell.Ink == canvas.InkGlow {
				t.Fatalf("daytime should be fully lit, found %+v", cell)
			}
		}
	}
}

func TestSkyKeepsItsOwnLight(t *testing.T) {
	// The moon, stars and snowflakes are drawn after the shadows.
	for _, season := range []scene.Season{scene.Summer, scene.Winter} {
		c := night(1, 10, season)
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				switch cell.Ink {
				case canvas.InkMoon, canvas.InkStar, canvas.InkFirefly:
					if cell.Shadow {
						t.Errorf("%s: sky cell %q should not be in shadow", season, cell.Rune)
					}
				}
			}
		}
	}
}

func TestLanternStaysPut(t *testing.T) {
	x0, y0, _ := find(night(1, 0, scene.Summer), canvas.InkLantern)
	for _, f := range []int{7, 55, 300} {
		if x, y, _ := find(night(1, f, scene.Summer), canvas.InkLantern); x != x0 || y != y0 {
			t.Errorf("frame %d: lantern moved from (%d,%d) to (%d,%d)", f, x0, y0, x, y)
		}
	}
}

func TestLanternHangsOnceTheBranchGrows(t *testing.T) {
	tr := grown(testSeed, 60, 20)
	sapling := placeLantern(tr, 1, 60)
	if sapling.hanger || sapling.y != tr.baseY {
		t.Errorf("before its branch grows, the lantern should sit on the soil: %+v", sapling)
	}
	full := placeLantern(tr, len(tr.marks), 60)
	if !full.hanger || full.y >= tr.baseY {
		t.Errorf("a grown tree should hold the lantern on a branch: %+v", full)
	}
	c := night(1, 0, scene.Summer)
	if c.At(full.x, full.y-1).Rune != ':' {
		t.Error("a hanging lantern needs its cord")
	}
}

func TestLanternFallsBackToTheOtherSide(t *testing.T) {
	// When the spot right of the trunk is taken, the lantern goes left.
	tr := tree{baseX: 10, baseY: 5, w: 20, used: make([]bool, 20*8)}
	tr.used[5*20+13] = true
	if l := placeLantern(tr, 0, 20); l.x != 7 {
		t.Errorf("lantern x = %d, want 7", l.x)
	}
}
