package arch

import (
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

const testSeed = 3

func render(w, h int, s scene.State) *canvas.Canvas {
	c := canvas.New(w, h)
	Draw(c, testSeed, s)
	return c
}

func TestGolden(t *testing.T) {
	for _, tt := range []struct {
		name string
		w, h int
		s    scene.State
	}{
		{"bare", 70, 22, scene.State{Growth: 0}},
		{"young", 70, 22, scene.State{Growth: 0.3}},
		{"summer", 70, 22, scene.State{Growth: 1, Frame: 30}},
		{"spring", 70, 22, scene.State{Growth: 1, Frame: 30, Season: scene.Spring}},
		{"autumn", 70, 22, scene.State{Growth: 1, Frame: 30, Season: scene.Autumn}},
		{"winter_night", 70, 22, scene.State{Growth: 1, Frame: 30, Season: scene.Winter, Night: true, MoonPhase: 0.3}},
		{"summer_night", 70, 22, scene.State{Growth: 0.8, Frame: 30, Night: true, MoonPhase: 0.6}},
		{"small", 30, 12, scene.State{Growth: 1}},
		{"plain", 14, 7, scene.State{Growth: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			golden.Assert(t, tt.name, render(tt.w, tt.h, tt.s).String())
		})
	}
}

func TestSceneInterface(t *testing.T) {
	var s scene.Scene = Scene{}
	if s.Name() != "arch" {
		t.Errorf("Name() = %q", s.Name())
	}
	c := canvas.New(50, 18)
	s.Draw(c, testSeed, scene.State{Growth: 1})
	if c.String() != render(50, 18, scene.State{Growth: 1}).String() {
		t.Error("Scene.Draw should match Draw")
	}
}

func inks(c *canvas.Canvas) map[canvas.Ink]int {
	n := map[canvas.Ink]int{}
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			n[cell.Ink]++
		}
	}
	return n
}

// static draws only what doesn't move, for growth comparisons.
func static(w, h int, s scene.State) *canvas.Canvas {
	c := canvas.New(w, h)
	drawStatic(c, testSeed, s)
	return c
}

// TestGrowingOnlyAdds: in every season and at several sizes, a higher
// growth keeps everything that already grew. Only the scenery may change:
// bare stone as moss and ivy cover it, and the far hills as the tree grows
// in front of them.
func TestGrowingOnlyAdds(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 40}, {40, 14}} {
		for _, season := range scene.Seasons {
			l := newLayout(testSeed, size[0], size[1])
			prev := static(size[0], size[1], scene.State{Season: season})
			for g := 0.05; g <= 1.0001; g += 0.05 {
				cur := static(size[0], size[1], scene.State{Growth: g, Season: season})
				for y := range cur.Height() {
					for x := range cur.Width() {
						p := prev.At(x, y)
						if p.Rune == ' ' || p.Ink == canvas.InkRockDim || y == l.hills() && l.open(x, y) {
							continue
						}
						if cur.At(x, y) != p {
							t.Fatalf("%v %s at %.2f: (%d,%d) changed from %q to %q", size, season, g, x, y, p.Rune, cur.At(x, y).Rune)
						}
					}
				}
				prev = cur
			}
		}
	}
}

func TestBareWall(t *testing.T) {
	c := render(70, 22, scene.State{Growth: 0})
	n := inks(c)
	if n[canvas.InkRock] == 0 || n[canvas.InkRockDim] == 0 {
		t.Errorf("a bare wall still has its stones and its arch: %v", n)
	}
	if n[canvas.InkWood]+n[canvas.InkLeaf] != 0 {
		t.Error("no tree before it starts growing")
	}
	for p := range newLayout(testSeed, 70, 22).ivy {
		if ink := c.At(p.x, p.y).Ink; ink != canvas.InkRockDim && ink != canvas.InkNone {
			t.Fatalf("ivy at (%d,%d) before anything grows", p.x, p.y)
		}
	}
}

// TestIvy: the ivy only ever grows on plain stone, spreads as growth goes
// up and frames the arch, climbing past where its curve starts.
func TestIvy(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 40}, {40, 14}, {200, 60}} {
		l := newLayout(testSeed, size[0], size[1])
		if len(l.ivy) == 0 {
			t.Fatalf("%v: no ivy", size)
		}
		framing := false
		for p := range l.ivy {
			if !l.wall(p.x, p.y) || l.ring(p.x, p.y) {
				t.Errorf("%v: ivy at (%d,%d) is off the stone", size, p.x, p.y)
			}
			if p.y < l.ys && p.x > l.cx-l.aw-3 && p.x < l.cx+l.aw+3 {
				framing = true
			}
		}
		if !framing {
			t.Errorf("%v: the ivy should frame the arch", size)
		}
	}
	shown := func(g float64) int {
		n := 0
		for _, cell := range newLayout(testSeed, 70, 22).ivy {
			if cell.at <= g {
				n++
			}
		}
		return n
	}
	if a, b, c := shown(0), shown(0.5), shown(1); a != 0 || b == 0 || c <= b {
		t.Errorf("the ivy should spread: %d, %d, %d cells", a, b, c)
	}
}

// TestIvyStemsSway: a stem leans now and then, but never zigzags.
func TestIvyStemsSway(t *testing.T) {
	for w := 30; w < 200; w += 11 {
		for h := 10; h < 60; h += 7 {
			l := newLayout(testSeed, w, h)
			for i, v := range l.vines(testSeed) {
				path := l.climb(i, v)
				for k := 2; k < len(path); k++ {
					a, b := path[k-1].x-path[k-2].x, path[k].x-path[k-1].x
					if path[k].y != path[k-1].y-1 || abs(b) > 1 || a != 0 && b == -a {
						t.Fatalf("%dx%d vine %d: bad step at (%d,%d)", w, h, i, path[k].x, path[k].y)
					}
				}
			}
		}
	}
}

func TestSeasonsOnTheIvy(t *testing.T) {
	ivyInks := func(season scene.Season) map[canvas.Ink]int {
		c := static(70, 22, scene.State{Growth: 1, Season: season})
		l := newLayout(testSeed, 70, 22)
		n := map[canvas.Ink]int{}
		for p := range l.ivy {
			n[c.At(p.x, p.y).Ink]++
		}
		return n
	}
	summer, autumn, winter := ivyInks(scene.Summer), ivyInks(scene.Autumn), ivyInks(scene.Winter)
	if summer[canvas.InkLeaf]+summer[canvas.InkLeafDark] == 0 {
		t.Error("green ivy in summer")
	}
	if autumn[canvas.InkAutumnRed] == 0 || autumn[canvas.InkLeaf] != 0 {
		t.Error("red ivy in autumn")
	}
	if winter[canvas.InkLeaf]+winter[canvas.InkLeafDark]+winter[canvas.InkAutumnRed] != 0 || winter[canvas.InkLeafDim] == 0 {
		t.Error("only bare stems in winter")
	}
	n := inks(render(70, 22, scene.State{Growth: 1, Season: scene.Winter}))
	if n[canvas.InkSnow] == 0 || n[canvas.InkGrass] != 0 {
		t.Error("snow on the wall and the grass in winter")
	}
}

// TestMoonInTheArch: on a clear night the moon hangs inside the arch;
// when the arch is too small for it, it goes back to its corner.
func TestMoonInTheArch(t *testing.T) {
	moon := func(w, h int) (in, out int) {
		c := render(w, h, scene.State{Growth: 1, Night: true, MoonPhase: 0.5})
		l := newLayout(testSeed, w, h)
		for y := range c.Height() {
			for x, cell := range c.Row(y) {
				if cell.Ink == canvas.InkMoon || cell.Ink == canvas.InkMoonDark {
					if l.open(x, y) {
						in++
					} else {
						out++
					}
				}
			}
		}
		return in, out
	}
	for _, size := range [][2]int{{70, 22}, {100, 40}, {200, 60}} {
		if in, out := moon(size[0], size[1]); in == 0 || out != 0 {
			t.Errorf("%v: %d moon cells in the arch, %d outside", size, in, out)
		}
	}
	if in, out := moon(24, 12); in != 0 || out == 0 {
		t.Errorf("a small arch has no room for the moon: %d in, %d out", in, out)
	}
}

// TestSkyStaysOffTheWall: stars never land on the wall or the ground, and
// clouds pass behind the wall.
func TestSkyStaysOffTheWall(t *testing.T) {
	l := newLayout(testSeed, 70, 22)
	for _, s := range []scene.State{
		{Growth: 1, Night: true, MoonPhase: 0.5},
		{Growth: 1, Frame: 100},
		{Growth: 1, Frame: 900},
	} {
		c := render(70, 22, s)
		for y := range c.Height() {
			for x, cell := range c.Row(y) {
				switch cell.Ink {
				case canvas.InkStar, canvas.InkStarDim, canvas.InkStarBright, canvas.InkCloud:
					if l.UnderCanopy(x, y, 0) {
						t.Errorf("%q at (%d,%d) is on the wall", cell.Rune, x, y)
					}
				}
			}
		}
	}
}

func TestButterflies(t *testing.T) {
	seen := func(s scene.State) bool {
		for f := 0; f < 60; f += 7 {
			s.Frame = f
			if inks(render(70, 22, s))[canvas.InkButterfly] > 0 {
				return true
			}
		}
		return false
	}
	if !seen(scene.State{Growth: 1}) || !seen(scene.State{Growth: 1, Season: scene.Spring}) {
		t.Error("butterflies on warm days in a grown meadow")
	}
	if seen(scene.State{Growth: butterfliesAt - 0.1}) || seen(scene.State{Growth: 1, Season: scene.Autumn}) ||
		seen(scene.State{Growth: 1, Night: true}) {
		t.Error("no butterflies early on, in autumn or at night")
	}
}

func TestFlowers(t *testing.T) {
	flowers := func(g float64) int {
		n := 0
		c := render(70, 22, scene.State{Growth: g})
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				if cell.Rune == '✿' {
					n++
				}
			}
		}
		return n
	}
	if flowers(flowersAt-0.05) != 0 || flowers(1) == 0 {
		t.Error("flowers open once the meadow has grown")
	}
}

func TestAnySizeIsSafe(_ *testing.T) {
	for w := 0; w < 90; w += 3 {
		for h := 0; h < 30; h++ {
			for _, season := range scene.Seasons {
				for _, f := range []int{0, 17, 400} {
					c := canvas.New(w, h)
					Draw(c, uint64(w*h+1), scene.State{Growth: 1, Frame: f, Season: season, Night: f == 17, MoonPhase: .4})
				}
			}
		}
	}
}

func TestDeterministic(t *testing.T) {
	s := scene.State{Growth: 0.8, Frame: 9, Night: true, MoonPhase: 0.3}
	if a, b := render(60, 20, s).String(), render(60, 20, s).String(); a != b {
		t.Error("same inputs drew different arches")
	}
}

// TestDoorwayShape: however tall the pane, the arch stays shaped like a
// doorway, at most about twice as tall as it is wide.
func TestDoorwayShape(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {82, 57}, {40, 80}, {200, 60}} {
		l := newLayout(testSeed, size[0], size[1])
		// Measured jamb to jamb, in rows: a cell is half as wide as tall.
		height, width := l.meadow-l.archTop+1, float64(2*l.aw+1)/2
		if float64(height) > 2*width {
			t.Errorf("%v: the opening is %d rows tall and only %.1f wide", size, height, width)
		}
	}
}
