package garden

import (
	"fmt"
	"math"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

const testSeed = 42

func render(w, h int, seed uint64, s State) string {
	c := canvas.New(w, h)
	Draw(c, seed, s)
	return c.String()
}

// TestGolden pins down how the garden looks at each stage. If you change the
// art on purpose, run `go test ./... -update` and review the diff.
func TestGolden(t *testing.T) {
	tests := []struct {
		name  string
		state State
	}{
		{"bare", State{Growth: 0}},
		{"sprouting", State{Growth: 0.2}},
		{"growing", State{Growth: 0.5}},
		{"budding", State{Growth: 0.75}},
		{"bloom_day", State{Growth: 1, Frame: 30}},
		{"bloom_night", State{Growth: 1, Frame: 30, Night: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			golden.Assert(t, tt.name, render(48, 12, testSeed, tt.state))
		})
	}
}

func TestDeterministic(t *testing.T) {
	s := State{Growth: 0.7, Frame: 12, Night: true}
	first, second := render(40, 10, 7, s), render(40, 10, 7, s)
	if first != second {
		t.Fatal("same inputs produced different gardens")
	}
}

func TestSeedChangesGarden(t *testing.T) {
	s := State{Growth: 1}
	if render(40, 10, 1, s) == render(40, 10, 2, s) {
		t.Fatal("different seeds produced identical gardens")
	}
}

func TestNightSkyAnimates(t *testing.T) {
	a := render(40, 10, testSeed, State{Growth: 1, Night: true, Frame: 0})
	b := render(40, 10, testSeed, State{Growth: 1, Night: true, Frame: 9})
	if a == b {
		t.Fatal("night sky did not change between frames")
	}
}

func TestGrowthAddsPlants(t *testing.T) {
	count := func(g float64) int {
		c := canvas.New(48, 12)
		Draw(c, testSeed, State{Growth: g})
		n := 0
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				if cell.Ink == canvas.InkStem || cell.Ink == canvas.InkLeaf {
					n++
				}
			}
		}
		return n
	}
	prev := -1
	for _, g := range []float64{0, 0.25, 0.5, 0.75, 1} {
		n := count(g)
		if n < prev {
			t.Errorf("growth %.2f has %d plant cells, fewer than before (%d)", g, n, prev)
		}
		prev = n
	}
	if count(0) != 0 {
		t.Error("bare garden should have no plants")
	}
	if count(1) == 0 {
		t.Error("full garden should have plants")
	}
}

func TestGrowthIsClamped(t *testing.T) {
	full := render(40, 10, testSeed, State{Growth: 1})
	bare := render(40, 10, testSeed, State{Growth: 0})
	if got := render(40, 10, testSeed, State{Growth: 5}); got != full {
		t.Error("growth above 1 should look like 1")
	}
	if got := render(40, 10, testSeed, State{Growth: -3}); got != bare {
		t.Error("growth below 0 should look like 0")
	}
	if got := render(40, 10, testSeed, State{Growth: math.NaN()}); got != bare {
		t.Error("NaN growth should look like 0")
	}
}

// TestAnySizeIsSafe draws every small size and many frames: nothing may
// panic and every row must stay inside the canvas.
func TestAnySizeIsSafe(t *testing.T) {
	for w := range 20 {
		for h := range 8 {
			for _, night := range []bool{false, true} {
				for frame := 0; frame < 200; frame += 7 {
					c := canvas.New(w, h)
					Draw(c, testSeed, State{Growth: 1, Frame: frame, Night: night})
					if c.Width() != w || c.Height() != h {
						t.Fatalf("canvas resized to %dx%d", c.Width(), c.Height())
					}
				}
			}
		}
	}
}

func TestDrawClearsPreviousContent(t *testing.T) {
	c := canvas.New(20, 6)
	c.Set(0, 0, 'X', canvas.InkStem)
	Draw(c, testSeed, State{})
	if c.At(0, 0).Rune == 'X' {
		t.Error("Draw should start from a blank canvas")
	}
}

func TestTriangle(t *testing.T) {
	var got []int
	for i := -2; i < 8; i++ {
		got = append(got, triangle(i, 4))
	}
	if want := "[2 1 0 1 2 3 2 1 0 1]"; fmt.Sprint(got) != want {
		t.Errorf("triangle sequence = %v, want %s", got, want)
	}
	if triangle(5, 1) != 0 || triangle(5, 0) != 0 {
		t.Error("triangle with n<=1 should be 0")
	}
}

func TestPlantStages(t *testing.T) {
	p := plant{x: 2, kind: daisy, maxH: 4, start: 0}
	tests := []struct {
		growth float64
		want   rune // rune at the top of the plant
	}{
		{0.1, ','}, // sprout
		{0.4, '|'}, // stem only
		{0.6, 'o'}, // bud
		{0.9, '✿'}, // bloom
	}
	for _, tt := range tests {
		c := canvas.New(5, 6)
		p.draw(c, 4, tt.growth)
		top := 4 - (p.height(tt.growth) - 1)
		if got := c.At(2, top).Rune; got != tt.want {
			t.Errorf("growth %.1f: top = %q, want %q\n%s", tt.growth, got, tt.want, c)
		}
	}
}

func TestBloomKinds(t *testing.T) {
	for _, k := range kinds {
		c := canvas.New(5, 6)
		plant{x: 2, kind: k, maxH: 4}.draw(c, 4, 1)
		top := c.At(2, 1)
		if top.Rune == '|' || top.Rune == ' ' {
			t.Errorf("kind %d has no head when fully grown:\n%s", k, c)
		}
	}
}

func TestPlantStartsLate(t *testing.T) {
	p := plant{maxH: 5, start: 0.5}
	if p.progress(0.5) != 0 || p.height(0.4) != 0 {
		t.Error("plant should not grow before its start")
	}
	if p.progress(1) != 1 {
		t.Error("plant should be fully grown at growth 1")
	}
}

func TestSceneInterface(t *testing.T) {
	s := Scene{}
	if s.Name() != "garden" {
		t.Errorf("Name() = %q", s.Name())
	}
	c := canvas.New(40, 10)
	s.Draw(c, testSeed, State{Growth: 1})
	if c.String() != render(40, 10, testSeed, State{Growth: 1}) {
		t.Error("Scene.Draw should match Draw")
	}
}
