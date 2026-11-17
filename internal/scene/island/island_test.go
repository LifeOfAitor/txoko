package island

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
		s    scene.State
	}{
		{"bare", scene.State{Growth: 0}},
		{"sapling", scene.State{Growth: 0.2}},
		{"flowing", scene.State{Growth: 0.6, Frame: 12}},
		{"summer", scene.State{Growth: 1, Frame: 30}},
		{"spring", scene.State{Growth: 1, Frame: 30, Season: scene.Spring}},
		{"autumn", scene.State{Growth: 1, Frame: 30, Season: scene.Autumn}},
		{"winter_night", scene.State{Growth: 1, Frame: 30, Season: scene.Winter, Night: true, MoonPhase: 0.3}},
		{"small", scene.State{Growth: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, h := 70, 24
			if tt.name == "small" {
				w, h = 30, 12
			}
			golden.Assert(t, tt.name, render(w, h, tt.s).String())
		})
	}
}

func TestSceneInterface(t *testing.T) {
	var s scene.Scene = Scene{}
	if s.Name() != "island" {
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
	drawIsland(c, testSeed, s)
	return c
}

// TestGrowingOnlyAdds: in every season, a higher growth keeps everything
// that was already on the island.
func TestGrowingOnlyAdds(t *testing.T) {
	for _, season := range scene.Seasons {
		prev := static(70, 24, scene.State{Season: season})
		for g := 0.1; g <= 1.0001; g += 0.1 {
			cur := static(70, 24, scene.State{Growth: g, Season: season})
			for y := range cur.Height() {
				for x := range cur.Width() {
					if p := prev.At(x, y); p.Rune != ' ' && cur.At(x, y) != p {
						t.Fatalf("%s at %.1f: (%d,%d) changed from %q to %q", season, g, x, y, p.Rune, cur.At(x, y).Rune)
					}
				}
			}
			prev = cur
		}
	}
}

func TestIslandIsAlwaysThere(t *testing.T) {
	n := inks(render(70, 24, scene.State{Growth: 0}))
	if n[canvas.InkRock] == 0 || n[canvas.InkRockDim] == 0 || n[canvas.InkSoil] == 0 {
		t.Errorf("a bare island still has rock and soil: %v", n)
	}
	if n[canvas.InkWood]+n[canvas.InkLeaf] != 0 {
		t.Error("no tree before it starts growing")
	}
}

func TestWaterfall(t *testing.T) {
	water := func(g float64, season scene.Season) int {
		n := inks(render(70, 24, scene.State{Growth: g, Season: season}))
		return n[canvas.InkWater] + n[canvas.InkWaterDim]
	}
	if water(waterfallAt-0.05, scene.Summer) != 0 {
		t.Error("no waterfall before the spring starts")
	}
	if a, b := water(0.55, scene.Summer), water(1, scene.Summer); a == 0 || b <= a {
		t.Errorf("the waterfall should appear and lengthen: %d then %d water cells", a, b)
	}
	frozen := render(70, 24, scene.State{Growth: 1, Season: scene.Winter})
	if inks(frozen)[canvas.InkWaterDim] != 0 {
		t.Error("in winter the waterfall freezes: no spray")
	}
	moving := func(f int) string { return render(70, 24, scene.State{Growth: 1, Frame: f}).String() }
	if moving(0) == moving(2) {
		t.Error("falling water should move")
	}
}

func TestDayAndNightSky(t *testing.T) {
	day := inks(render(70, 24, scene.State{Growth: 1}))
	night := inks(render(70, 24, scene.State{Growth: 1, Night: true, MoonPhase: 0.5}))
	if day[canvas.InkCloud] == 0 || night[canvas.InkCloud] != 0 {
		t.Error("clouds drift by day, the sky is clear at night")
	}
	if night[canvas.InkMoon] == 0 || night[canvas.InkStarDim]+night[canvas.InkStar] == 0 {
		t.Error("night brings the moon and the stars")
	}
	if day[canvas.InkMoon] != 0 {
		t.Error("no moon by day")
	}
}

// TestSkyStaysOffTheIsland: stars and the moon never land on the island,
// its tree or its waterfall.
func TestSkyStaysOffTheIsland(t *testing.T) {
	c := render(70, 24, scene.State{Growth: 1, Night: true, MoonPhase: 0.5})
	l := newLayout(testSeed, 70, 24)
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			switch cell.Ink {
			case canvas.InkStar, canvas.InkStarDim, canvas.InkStarBright, canvas.InkMoon:
				if l.UnderCanopy(x, y, 0) {
					t.Errorf("sky %q at (%d,%d) is on the island", cell.Rune, x, y)
				}
			}
		}
	}
}

func TestBirds(t *testing.T) {
	birds := func(g float64, season scene.Season) bool {
		for f := 0; f < 40; f += 7 {
			c := render(70, 24, scene.State{Growth: g, Frame: f, Season: season})
			for y := range c.Height() {
				for _, cell := range c.Row(y) {
					if cell.Rune == 'v' || cell.Rune == '^' {
						return true
					}
				}
			}
		}
		return false
	}
	if !birds(1, scene.Summer) {
		t.Error("birds should visit a grown island")
	}
	if birds(0.4, scene.Summer) || birds(1, scene.Winter) {
		t.Error("no birds on a young island or in winter")
	}
}

func TestSeasonsOnTheGrass(t *testing.T) {
	spring := inks(render(70, 24, scene.State{Growth: 1, Season: scene.Spring}))
	autumn := inks(render(70, 24, scene.State{Growth: 1, Season: scene.Autumn}))
	winter := inks(render(70, 24, scene.State{Growth: 1, Season: scene.Winter}))
	if spring[canvas.InkBlossom] == 0 {
		t.Error("spring brings blossom")
	}
	if autumn[canvas.InkAutumnRed]+autumn[canvas.InkAutumnOrange]+autumn[canvas.InkAutumnGold] == 0 {
		t.Error("autumn brings warm colors")
	}
	if winter[canvas.InkSnow] == 0 || winter[canvas.InkGrass] != 0 {
		t.Error("winter covers the grass with snow")
	}
}

func TestAnySizeIsSafe(_ *testing.T) {
	for w := 0; w < 60; w += 3 {
		for h := 0; h < 20; h++ {
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
		t.Error("same inputs drew different islands")
	}
}
