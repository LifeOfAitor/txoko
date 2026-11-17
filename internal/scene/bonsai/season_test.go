package bonsai

import (
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

func TestSeasonGolden(t *testing.T) {
	for _, tt := range []struct {
		name  string
		state scene.State
	}{
		{"spring", scene.State{Growth: 1, Frame: 40, Season: scene.Spring}},
		{"autumn", scene.State{Growth: 1, Frame: 40, Season: scene.Autumn}},
		{"winter", scene.State{Growth: 1, Frame: 40, Season: scene.Winter}},
		{"winter_night", scene.State{Growth: 1, Frame: 40, Season: scene.Winter, Night: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			golden.Assert(t, tt.name, render(60, 20, testSeed, tt.state))
		})
	}
}

// inks counts how many cells use each ink.
func inks(c *canvas.Canvas) map[canvas.Ink]int {
	n := map[canvas.Ink]int{}
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			n[cell.Ink]++
		}
	}
	return n
}

func treeOnly(season scene.Season, g float64) *canvas.Canvas {
	c := canvas.New(60, 20)
	drawTree(c, testSeed, scene.State{Growth: g, Season: season})
	return c
}

func TestSeasonsDressTheTree(t *testing.T) {
	leaves := func(season scene.Season) int {
		n := 0
		for _, m := range grown(testSeed, 60, 20).marks {
			if _, _, onTree := dressLeaf(m, season); m.leaf && onTree {
				n++
			}
		}
		return n
	}

	spring := inks(treeOnly(scene.Spring, 1))
	if spring[canvas.InkBlossom] == 0 || spring[canvas.InkBlossomLight] == 0 {
		t.Error("spring should be in blossom")
	}
	if spring[canvas.InkBlossom] < spring[canvas.InkLeaf] {
		t.Error("spring should be mostly blossom, not green")
	}

	autumn := inks(treeOnly(scene.Autumn, 1))
	for _, ink := range []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold} {
		if autumn[ink] == 0 {
			t.Errorf("autumn is missing ink %d", ink)
		}
	}
	if autumn[canvas.InkLeaf] != 0 {
		t.Error("no bright summer green in autumn")
	}

	winter := inks(treeOnly(scene.Winter, 1))
	if winter[canvas.InkSnow] == 0 {
		t.Error("winter should have snow")
	}
	if leaves(scene.Winter) >= leaves(scene.Summer)/2 {
		t.Errorf("winter should be mostly bare: %d leaves vs %d in summer", leaves(scene.Winter), leaves(scene.Summer))
	}
	if leaves(scene.Autumn) >= leaves(scene.Summer) {
		t.Error("some leaves should have fallen in autumn")
	}
}

// TestSeasonalGrowthOnlyAdds: in every season, growing never removes or
// recolors what was already drawn (weather aside).
func TestSeasonalGrowthOnlyAdds(t *testing.T) {
	for _, season := range scene.Seasons {
		prev := treeOnly(season, 0)
		for g := 0.1; g <= 1.0001; g += 0.1 {
			cur := treeOnly(season, g)
			for y := range cur.Height() {
				for x := range cur.Width() {
					p := prev.At(x, y)
					if p.Rune != ' ' && cur.At(x, y) != p {
						t.Fatalf("%s at growth %.1f changed (%d,%d) %q -> %q", season, g, x, y, p.Rune, cur.At(x, y).Rune)
					}
				}
			}
			prev = cur
		}
	}
}

func TestWeather(t *testing.T) {
	frame := func(season scene.Season, g float64, f int) string {
		return render(60, 20, testSeed, scene.State{Growth: g, Frame: f, Season: season})
	}
	for _, season := range scene.Seasons {
		if frame(season, 1, 0) == frame(season, 1, 25) {
			t.Errorf("%s weather should move", season)
		}
	}
	// Snow falls even on a bare pot; petals and leaves need a tree first.
	if frame(scene.Winter, 0, 0) == frame(scene.Winter, 0, 25) {
		t.Error("snow should fall even before the tree grows")
	}
	if frame(scene.Spring, 0.1, 0) != frame(scene.Spring, 0.1, 25) {
		t.Error("a sapling has no petals to drop yet")
	}
}

func TestFirefliesOnlyOnWarmNights(t *testing.T) {
	has := func(season scene.Season) bool {
		c := canvas.New(60, 20)
		for f := 0; f < 60; f += 7 {
			Draw(c, testSeed, scene.State{Growth: 1, Frame: f, Night: true, Season: season})
			if inks(c)[canvas.InkFirefly] > 0 {
				return true
			}
		}
		return false
	}
	if !has(scene.Summer) || !has(scene.Spring) {
		t.Error("summer and spring nights should have fireflies")
	}
	if has(scene.Autumn) || has(scene.Winter) {
		t.Error("no fireflies on cold nights")
	}
}

func TestSeasonsAreSafeAtAnySize(_ *testing.T) {
	for _, season := range scene.Seasons {
		for w := 0; w < 40; w += 3 {
			for h := range 12 {
				c := canvas.New(w, h)
				for _, f := range []int{0, 13, 99} {
					Draw(c, 7, scene.State{Growth: 1, Frame: f, Night: true, Season: season})
				}
			}
		}
	}
}

func TestNoiseIsStableAndUniform(t *testing.T) {
	if a, b := noise(3, 4, 1), noise(3, 4, 1); a != b {
		t.Fatal("noise must be deterministic")
	}
	below := 0
	for i := range 10000 {
		v := noise(i%100, i/100, 5)
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
