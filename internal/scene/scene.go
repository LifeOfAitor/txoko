// Package scene defines what every txoko scene has in common.
//
// A scene is a pure drawing function: given a canvas, a seed and a State it
// paints the same picture every time. Scenes know nothing about terminals or
// colors — they paint semantic inks that a theme later turns into styles.
package scene

import "github.com/LifeOfAitor/txoko/internal/canvas"

// State is everything that changes the picture besides the seed and size.
type State struct {
	// Growth is overall progress from 0 (nothing yet) to 1 (fully grown).
	// Scenes clamp values outside that range.
	Growth float64
	// Frame is an ever-increasing animation counter.
	Frame int
	// Night switches the scene to its night look.
	Night bool
	// Season dresses the scene for the time of year.
	Season Season
}

// Scene is a drawable theme such as the bonsai or the garden.
type Scene interface {
	// Name is the short identifier used in flags and the UI, e.g. "bonsai".
	Name() string
	// Draw paints the scene onto c, replacing whatever was there.
	Draw(c *canvas.Canvas, seed uint64, s State)
}
