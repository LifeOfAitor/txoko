// Package bonsai draws txoko's flagship scene: a bonsai tree in a pot that
// really grows. The tree, the sky and the seasons come from package nature;
// this package adds the pot and what each season leaves on its soil.
package bonsai

import (
	"math/rand/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
)

// potHeight is the number of rows the pot takes at the bottom.
const potHeight = 2

// Scene exposes the bonsai through the scene.Scene interface.
type Scene struct{}

// Name implements scene.Scene.
func (Scene) Name() string { return "bonsai" }

// Draw implements scene.Scene.
func (Scene) Draw(c *canvas.Canvas, seed uint64, s scene.State) { Draw(c, seed, s) }

// Draw paints the bonsai onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, s scene.State) {
	t, leaves, g, ok := drawTree(c, seed, s)
	if !ok {
		return
	}
	if s.Night {
		nature.DimForNight(c)
		nature.DrawSky(c, seed, t, s.MoonPhase, s.Frame)
		nature.DrawFireflies(c, t.BaseY, leaves, g, s.Frame, s.Season)
	}
	nature.DrawWeather(c, seed, t.BaseY, leaves, g, s.Frame, s.Season)
}

// drawTree paints everything that doesn't move: pot, tree and what the
// season leaves on them. It returns the tree, its visible leaves and the
// clamped growth for the animations, or ok=false when there's no room.
func drawTree(c *canvas.Canvas, seed uint64, s scene.State) (t nature.Tree, leaves []nature.Mark, g float64, ok bool) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w == 0 || h == 0 {
		return nature.Tree{}, nil, 0, false
	}
	drawPot(c, seed)
	if h <= potHeight {
		return nature.Tree{}, nil, 0, false
	}

	t = nature.Grow(seed, w, h)
	g = nature.Clamp01(s.Growth)
	visible, leaves := nature.Reveal(c, t, g, s.Season, 0)
	dressGround(c, t, visible, s.Season)
	return t, leaves, g, true
}

// drawPot draws a shallow pot centered at the bottom of the canvas:
//
//	\_.~_:__.,_~_/
//	 \__________/
func drawPot(c *canvas.Canvas, seed uint64) {
	w, h := c.Width(), c.Height()
	pw := min(max(w*2/5, 7), 36, w)
	x0 := (w - pw) / 2
	rng := rand.New(rand.NewPCG(seed, 0x9075))

	soil := []rune("___.__:_~__.,_")
	rim := h - potHeight
	for x := x0 + 1; x < x0+pw-1; x++ {
		r := soil[rng.IntN(len(soil))]
		ink := canvas.InkPot
		if r != '_' {
			ink = canvas.InkSoil
		}
		c.Set(x, rim, r, ink)
	}
	c.Set(x0, rim, '\\', canvas.InkPot)
	c.Set(x0+pw-1, rim, '/', canvas.InkPot)

	base := h - 1
	for x := x0 + 2; x < x0+pw-2; x++ {
		c.Set(x, base, '_', canvas.InkPot)
	}
	c.Set(x0+1, base, '\\', canvas.InkPot)
	c.Set(x0+pw-2, base, '/', canvas.InkPot)
}

// dressGround adds what a season leaves around the tree: fallen autumn
// leaves on the soil, snow on the soil and on top of the branches.
func dressGround(c *canvas.Canvas, t nature.Tree, visible []nature.Mark, season scene.Season) {
	rim := c.Height() - potHeight
	switch season {
	case scene.Autumn:
		for x := range c.Width() {
			if cell := c.At(x, rim); cell.Ink == canvas.InkSoil || cell.Ink == canvas.InkPot && cell.Rune == '_' {
				if v := nature.Noise(x, rim, 21); v < 0.35 {
					inks := []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}
					c.Set(x, rim, ',', inks[int(v*100)%len(inks)])
				}
			}
		}
	case scene.Winter:
		for x := range c.Width() {
			if cell := c.At(x, rim); cell.Ink == canvas.InkSoil || cell.Ink == canvas.InkPot && cell.Rune == '_' {
				c.Set(x, rim, []rune{'_', '.', '_'}[x%3], canvas.InkSnow)
			}
		}
		nature.SnowOnBranches(c, t, visible, 0)
	}
}
