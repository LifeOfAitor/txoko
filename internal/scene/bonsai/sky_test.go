package bonsai

import (
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

func sky(w, h int, phase float64, frame int) (*canvas.Canvas, tree) {
	c := canvas.New(w, h)
	Draw(c, testSeed, scene.State{Growth: 1, Night: true, Frame: frame, MoonPhase: phase})
	return c, grown(testSeed, w, h)
}

// moonCells splits the moon into its lit and dark cells' x positions.
func moonCells(c *canvas.Canvas) (lit, dark []int) {
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			switch {
			case cell.Ink == canvas.InkMoon && cell.Rune != '☾':
				lit = append(lit, x)
			case cell.Ink == canvas.InkMoonDark:
				dark = append(dark, x)
			}
		}
	}
	return lit, dark
}

func mean(xs []int) float64 {
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return float64(sum) / float64(len(xs))
}

func TestMoonPhases(t *testing.T) {
	newMoon, _ := sky(80, 26, 0, 0)
	if lit, dark := moonCells(newMoon); len(lit) != 0 || len(dark) == 0 {
		t.Errorf("new moon: %d lit, %d dark cells; want only the faint disc", len(lit), len(dark))
	}

	full, _ := sky(80, 26, 0.5, 0)
	if lit, dark := moonCells(full); len(lit) == 0 || len(dark) != 0 {
		t.Errorf("full moon: %d lit, %d dark cells; want fully lit", len(lit), len(dark))
	}

	// First quarter is lit on the right, last quarter on the left.
	first, _ := sky(80, 26, 0.25, 0)
	lit, dark := moonCells(first)
	if len(lit) == 0 || len(dark) == 0 || mean(lit) <= mean(dark) {
		t.Errorf("first quarter should be lit on the right: lit x̄ %.1f, dark x̄ %.1f", mean(lit), mean(dark))
	}
	last, _ := sky(80, 26, 0.75, 0)
	lit, dark = moonCells(last)
	if len(lit) == 0 || len(dark) == 0 || mean(lit) >= mean(dark) {
		t.Errorf("last quarter should be lit on the left: lit x̄ %.1f, dark x̄ %.1f", mean(lit), mean(dark))
	}
}

func TestMoonGrowsWithTheScreenAndAvoidsTheCrown(t *testing.T) {
	size := func(w, h int) int {
		c, tr := sky(w, h, 0.5, 0)
		lit, _ := moonCells(c)
		for y := range c.Height() {
			for x, cell := range c.Row(y) {
				if cell.Ink == canvas.InkMoon && tr.underCanopy(x, y, 0) {
					t.Errorf("%dx%d: the moon overlaps the crown at (%d,%d)", w, h, x, y)
				}
			}
		}
		return len(lit)
	}
	small, large := size(40, 16), size(100, 30)
	if large <= small {
		t.Errorf("the moon should be bigger on a bigger screen: %d vs %d cells", large, small)
	}
	if large < 30 {
		t.Errorf("a wide screen deserves a big moon, got %d cells", large)
	}
}

func TestTinySkyFallsBackToASymbol(t *testing.T) {
	c, _ := sky(16, 8, 0.5, 0)
	found := false
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			found = found || cell.Rune == '☾'
		}
	}
	if !found {
		t.Error("a tiny sky should still show a small moon")
	}
}

func TestStarsComeInFourSizesOutsideTheCrown(t *testing.T) {
	c, tr := sky(100, 30, 0.5, 0)
	n := map[canvas.Ink]int{}
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			switch cell.Ink {
			case canvas.InkStarDim, canvas.InkStar, canvas.InkStarBright:
				n[cell.Ink]++
				if tr.underCanopy(x, y, 0) {
					t.Errorf("star %q below the tree's silhouette at (%d,%d)", cell.Rune, x, y)
				}
			}
		}
	}
	if n[canvas.InkStarDim] == 0 || n[canvas.InkStar] == 0 || n[canvas.InkStarBright] == 0 {
		t.Errorf("want faint, normal and bright stars: %v", n)
	}
	if total := n[canvas.InkStarDim] + n[canvas.InkStar] + n[canvas.InkStarBright]; total < 60 { // the old sky had ~22
		t.Errorf("the sky should be full of stars, got %d", total)
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

func TestShootingStar(t *testing.T) {
	visible := func(frame int) bool {
		c, _ := sky(80, 26, 0.5, frame)
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				if cell.Rune == '.' && cell.Ink == canvas.InkStar {
					return true // only the shooting star's trail uses '.'
				}
			}
		}
		return false
	}
	if !visible(shootingEvery) || !visible(shootingEvery+5) {
		t.Error("a shooting star should cross the sky at the start of each cycle")
	}
	if visible(shootingEvery + shootingFor + 10) {
		t.Error("the shooting star should be gone after its flight")
	}
}

// TestWideShortPaneHasSkyAboveTheTree: in a wide, short pane the stars
// must not be pushed to the sides only.
func TestWideShortPaneHasSkyAboveTheTree(t *testing.T) {
	c, tr := sky(140, 18, 0.5, 0)
	above := 0
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			isStar := cell.Ink == canvas.InkStarDim || cell.Ink == canvas.InkStar || cell.Ink == canvas.InkStarBright
			if isStar && x >= tr.left && x <= tr.right && y < tr.top {
				above++
			}
		}
	}
	if above == 0 {
		t.Error("no stars above the tree in a wide, short pane")
	}
	if tr.top < 2 {
		t.Errorf("the tree should leave the top rows to the sky, its top is row %d", tr.top)
	}
}

func TestUnderCanopy(t *testing.T) {
	tr := grown(testSeed, 60, 20)
	if tr.underCanopy(-1, 5, 0) || tr.underCanopy(60, 5, 0) {
		t.Error("columns outside the canvas are never under the canopy")
	}
	if !tr.underCanopy(tr.baseX, tr.baseY, 0) {
		t.Error("the trunk's base is under the canopy")
	}
	if tr.underCanopy(tr.baseX, 0, 0) && tr.top > 0 {
		t.Error("the top row is sky when the tree doesn't reach it")
	}
}

func TestBoxContains(t *testing.T) {
	b := box{2, 2, 4, 3}
	if !b.contains(2, 2, 0) || b.contains(1, 2, 0) || !b.contains(1, 2, 1) {
		t.Error("box.contains is wrong")
	}
}
