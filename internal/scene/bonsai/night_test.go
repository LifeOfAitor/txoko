package bonsai

import (
	"testing"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

func night(season scene.Season) *canvas.Canvas {
	c := canvas.New(60, 20)
	Draw(c, testSeed, scene.State{Growth: 1, Frame: 10, Night: true, Season: season})
	return c
}

// TestNightDimsTheWholeTree: at night the tree, the pot and the ground are
// evenly dimmed — no half-lit pot — while the sky keeps its own light.
func TestNightDimsTheWholeTree(t *testing.T) {
	for _, season := range scene.Seasons {
		day := canvas.New(60, 20)
		drawTree(day, testSeed, scene.State{Growth: 1, Season: season})
		c := night(season)
		for y := range c.Height() {
			for x, cell := range c.Row(y) {
				if day.IsEmpty(x, y) {
					continue // not part of the tree, pot or ground
				}
				if cell == day.At(x, y) && !cell.Shadow {
					t.Fatalf("%s night: %q at (%d,%d) should be dimmed", season, cell.Rune, x, y)
				}
			}
		}
	}
}

func TestNightSkyKeepsItsLight(t *testing.T) {
	for _, season := range scene.Seasons {
		c := night(season)
		for y := range c.Height() {
			for _, cell := range c.Row(y) {
				switch cell.Ink {
				case canvas.InkMoon, canvas.InkStar, canvas.InkFirefly:
					if cell.Shadow {
						t.Errorf("%s: %q in the sky should not be dimmed", season, cell.Rune)
					}
				}
			}
		}
	}
}

func TestDayIsNotDimmed(t *testing.T) {
	c := canvas.New(60, 20)
	Draw(c, testSeed, scene.State{Growth: 1})
	for y := range c.Height() {
		for _, cell := range c.Row(y) {
			if cell.Shadow {
				t.Fatalf("daytime cell %q is dimmed", cell.Rune)
			}
		}
	}
}
