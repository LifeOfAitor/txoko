package bonsai

import (
	"math"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

const testSeed = 42

func render(w, h int, seed uint64, s scene.State) string {
	c := canvas.New(w, h)
	Draw(c, seed, s)
	return c.String()
}

// TestGolden pins down how the bonsai looks while it grows. If you change
// the art on purpose, run `go test ./... -update` and review the diff.
func TestGolden(t *testing.T) {
	tests := []struct {
		name  string
		w, h  int
		state scene.State
	}{
		{"seed", 60, 20, scene.State{Growth: 0}},
		{"sapling", 60, 20, scene.State{Growth: 0.15}},
		{"young", 60, 20, scene.State{Growth: 0.4}},
		{"grown", 60, 20, scene.State{Growth: 0.75}},
		{"full_day", 60, 20, scene.State{Growth: 1, Frame: 40}},
		{"full_night", 60, 20, scene.State{Growth: 1, Frame: 40, Night: true}},
		{"small", 30, 10, scene.State{Growth: 1}},
		{"tiny", 8, 5, scene.State{Growth: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			golden.Assert(t, tt.name, render(tt.w, tt.h, testSeed, tt.state))
		})
	}
}

func TestSceneInterface(t *testing.T) {
	var s scene.Scene = Scene{}
	if s.Name() != "bonsai" {
		t.Errorf("Name() = %q", s.Name())
	}
	c := canvas.New(40, 12)
	s.Draw(c, testSeed, scene.State{Growth: 1})
	if c.String() != render(40, 12, testSeed, scene.State{Growth: 1}) {
		t.Error("Scene.Draw should match Draw")
	}
}

func TestDeterministic(t *testing.T) {
	s := scene.State{Growth: 0.7, Frame: 12, Night: true}
	first, second := render(50, 16, 7, s), render(50, 16, 7, s)
	if first != second {
		t.Fatal("same inputs produced different trees")
	}
}

func TestSeedChangesTree(t *testing.T) {
	s := scene.State{Growth: 1}
	if render(50, 16, 1, s) == render(50, 16, 2, s) {
		t.Fatal("different seeds produced identical trees")
	}
}

// TestGrowthOnlyAdds checks that growing never removes or changes a cell
// that was already there: a higher growth shows a superset of the tree.
func TestGrowthOnlyAdds(t *testing.T) {
	const w, h = 60, 20
	prev := canvas.New(w, h)
	Draw(prev, testSeed, scene.State{Growth: 0})
	for g := 0.05; g <= 1.0001; g += 0.05 {
		cur := canvas.New(w, h)
		Draw(cur, testSeed, scene.State{Growth: g})
		for y := range h {
			for x := range w {
				p := prev.At(x, y)
				if p.Rune == ' ' || isFallingLeaf(p) {
					continue // falling leaves are animation, not tree
				}
				if cur.At(x, y) != p {
					t.Fatalf("growth %.2f changed (%d,%d) from %q to %q", g, x, y, p.Rune, cur.At(x, y).Rune)
				}
			}
		}
		prev = cur
	}
}

func isFallingLeaf(c canvas.Cell) bool {
	return c.Ink == canvas.InkLeafDim && (c.Rune == ',' || c.Rune == '`')
}

func TestGrowthRevealsEvenly(t *testing.T) {
	tr := nature.Grow(testSeed, 60, 20)
	if len(tr.Marks) < 50 {
		t.Fatalf("tree has only %d marks", len(tr.Marks))
	}
	for i := 1; i < len(tr.Marks); i++ {
		if tr.Marks[i].Tick < tr.Marks[i-1].Tick {
			t.Fatalf("marks out of growth order at %d", i)
		}
	}
	count := func(g float64) int {
		c := canvas.New(60, 20)
		Draw(c, testSeed, scene.State{Growth: g})
		n := 0
		for y := range c.Height() - potHeight {
			for _, cell := range c.Row(y) {
				if cell.Ink != canvas.InkNone {
					n++
				}
			}
		}
		return n
	}
	half, full := count(0.5), count(1)
	if half < full*4/10 || half > full*6/10 {
		t.Errorf("half growth shows %d of %d cells, want about half", half, full)
	}
}

// TestTallPaneKeepsProportions: in a tall, narrow pane the tree must stay
// tree-shaped instead of growing a trunk as tall as the screen.
func TestTallPaneKeepsProportions(t *testing.T) {
	const w, h = 58, 70
	for _, seed := range []uint64{1, 2, 3, testSeed, 99, 12345} {
		tr := nature.Grow(seed, w, h)
		top := h
		for _, m := range tr.Marks {
			top = min(top, m.Y)
		}
		// About as tall as half the width looks like a tree; much more is a pole.
		if treeH := tr.BaseY - top + 1; treeH > w/2 {
			t.Errorf("seed %d: tree is %d rows tall in a %dx%d pane, want at most %d", seed, treeH, w, h, w/2)
		}
	}
}

func TestTreeHasWoodAndLeaves(t *testing.T) {
	c := canvas.New(60, 20)
	Draw(c, testSeed, scene.State{Growth: 1})
	seen := map[canvas.Ink]bool{}
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			seen[cell.Ink] = true
		}
	}
	for _, ink := range []canvas.Ink{
		canvas.InkWood, canvas.InkWoodDim,
		canvas.InkLeaf, canvas.InkLeafDark, canvas.InkLeafDim,
		canvas.InkPot, canvas.InkSoil,
	} {
		if !seen[ink] {
			t.Errorf("full tree is missing ink %d", ink)
		}
	}
}

func TestBareTreeShowsOnlyThePot(t *testing.T) {
	c := canvas.New(40, 12)
	Draw(c, testSeed, scene.State{Growth: 0})
	for y := range c.Height() - potHeight {
		for x := range c.Width() {
			if !c.IsEmpty(x, y) {
				t.Fatalf("growth 0 drew %q at (%d,%d)", c.At(x, y).Rune, x, y)
			}
		}
	}
}

func TestGrowthIsClamped(t *testing.T) {
	full := render(40, 12, testSeed, scene.State{Growth: 1})
	bare := render(40, 12, testSeed, scene.State{Growth: 0})
	if render(40, 12, testSeed, scene.State{Growth: 3}) != full {
		t.Error("growth above 1 should look like 1")
	}
	if render(40, 12, testSeed, scene.State{Growth: -1}) != bare {
		t.Error("growth below 0 should look like 0")
	}
	if render(40, 12, testSeed, scene.State{Growth: math.NaN()}) != bare {
		t.Error("NaN growth should look like 0")
	}
}

func TestAnimations(t *testing.T) {
	night := func(frame int) string {
		return render(60, 20, testSeed, scene.State{Growth: 1, Night: true, Frame: frame})
	}
	if night(0) == night(9) {
		t.Error("night scene did not animate")
	}
	day := func(frame int) string {
		return render(60, 20, testSeed, scene.State{Growth: 1, Frame: frame})
	}
	if day(0) == day(20) {
		t.Error("falling leaves did not animate")
	}
	young := func(frame int) string {
		return render(60, 20, testSeed, scene.State{Growth: 0.3, Frame: frame})
	}
	if young(0) != young(20) {
		t.Error("a young day tree should be still (no falling leaves yet)")
	}
}

// TestAnySizeIsSafe draws many sizes, growths and frames: nothing may panic.
func TestAnySizeIsSafe(_ *testing.T) {
	for w := 0; w < 45; w += 3 {
		for h := range 14 {
			for _, g := range []float64{0, 0.5, 1} {
				for frame := 0; frame < 300; frame += 37 {
					for _, night := range []bool{false, true} {
						c := canvas.New(w, h)
						Draw(c, uint64(w*h+1), scene.State{Growth: g, Frame: frame, Night: night})
					}
				}
			}
		}
	}
}
