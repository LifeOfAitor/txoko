package bonsai

import "github.com/LifeOfAitor/txoko/internal/canvas"

// dimForNight puts the whole tree, the pot and the ground in shadow, so the
// night scene is soft and calm. Call it after the tree and before the sky:
// the moon, the stars, fireflies and whatever is falling through the air
// are drawn afterwards and keep their own light.
func dimForNight(c *canvas.Canvas) {
	for y := range c.Height() {
		for x := range c.Width() {
			c.Shade(x, y)
		}
	}
}
