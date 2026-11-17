// Package canvas provides a fixed-size grid of characters that scenes draw
// into. Each cell holds a rune and an Ink — a semantic color role such as
// "stem" or "star" — rather than a concrete color. Turning inks into real
// terminal colors is the job of a theme, which keeps scenes free of styling
// and easy to test as plain text.
package canvas

import "strings"

// Ink is the semantic color role of a cell.
type Ink uint8

// Inks known to txoko. InkNone means "terminal default".
const (
	InkNone Ink = iota
	InkSoil
	InkGrass
	InkStem
	InkLeaf
	InkBud
	InkPetalPink
	InkPetalRed
	InkPetalYellow
	InkCenter
	InkStar
	InkMoon
	InkSun
	InkCloud
	InkFirefly
	InkButterfly
	InkWood
	InkWoodDim
	InkLeafDark
	InkLeafDim
	InkPot
	InkBlossom      // cherry blossom pink
	InkBlossomLight // pale petals
	InkAutumnRed
	InkAutumnOrange
	InkAutumnGold
	InkSnow
	InkStarDim    // the faintest, most distant stars
	InkStarBright // the brightest stars
	InkMoonDark   // the unlit part of the moon

	// InkCount is the number of inks. Keep it last.
	InkCount
)

// Cell is a single character on the canvas.
type Cell struct {
	Rune rune
	Ink  Ink
	// Shadow marks a cell that is out of the light, e.g. the tree at
	// night. Themes draw it dimmed.
	Shadow bool
}

var blank = Cell{Rune: ' '}

// Canvas is a width × height grid of cells, addressed from the top-left
// corner (0, 0). Every method is safe to call with out-of-bounds coordinates:
// writes are ignored and reads return a blank cell. That lets scenes draw
// freely near the edges without bounds checks everywhere.
type Canvas struct {
	w, h  int
	cells []Cell
}

// New returns a blank canvas. Negative sizes are treated as zero.
func New(w, h int) *Canvas {
	w, h = max(w, 0), max(h, 0)
	c := &Canvas{w: w, h: h, cells: make([]Cell, w*h)}
	c.Clear()
	return c
}

// Resize changes the canvas size in place, clearing it. Keeping the same
// *Canvas lets a UI hold on to it while its layout changes.
func (c *Canvas) Resize(w, h int) {
	w, h = max(w, 0), max(h, 0)
	if w == c.w && h == c.h {
		return
	}
	c.w, c.h = w, h
	c.cells = make([]Cell, w*h)
	c.Clear()
}

// Width returns the number of columns.
func (c *Canvas) Width() int { return c.w }

// Height returns the number of rows.
func (c *Canvas) Height() int { return c.h }

// Clear resets every cell to a blank space.
func (c *Canvas) Clear() {
	for i := range c.cells {
		c.cells[i] = blank
	}
}

// In reports whether (x, y) is inside the canvas.
func (c *Canvas) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.w && y < c.h
}

// Set writes a rune with the given ink at (x, y).
func (c *Canvas) Set(x, y int, r rune, ink Ink) {
	if c.In(x, y) {
		c.cells[y*c.w+x] = Cell{Rune: r, Ink: ink}
	}
}

// SetIfEmpty writes at (x, y) only if that cell is blank, so background
// details never paint over foreground ones. It reports whether it wrote.
func (c *Canvas) SetIfEmpty(x, y int, r rune, ink Ink) bool {
	if !c.IsEmpty(x, y) {
		return false
	}
	c.Set(x, y, r, ink)
	return true
}

// Shade puts the cell at (x, y) in shadow. Blank cells are left alone:
// there's nothing there to darken.
func (c *Canvas) Shade(x, y int) {
	if c.In(x, y) && !c.IsEmpty(x, y) {
		c.cells[y*c.w+x].Shadow = true
	}
}

// IsEmpty reports whether (x, y) is inside the canvas and blank.
func (c *Canvas) IsEmpty(x, y int) bool {
	return c.In(x, y) && c.cells[y*c.w+x] == blank
}

// At returns the cell at (x, y), or a blank cell when out of bounds.
func (c *Canvas) At(x, y int) Cell {
	if !c.In(x, y) {
		return blank
	}
	return c.cells[y*c.w+x]
}

// Row returns the cells of row y. The slice aliases the canvas and must not
// be modified. It returns nil when y is out of bounds.
func (c *Canvas) Row(y int) []Cell {
	if y < 0 || y >= c.h {
		return nil
	}
	return c.cells[y*c.w : (y+1)*c.w]
}

// String renders the canvas as plain text, one line per row, with trailing
// spaces trimmed. It is meant for tests and debugging.
func (c *Canvas) String() string {
	var b strings.Builder
	for y := range c.h {
		var line strings.Builder
		for _, cell := range c.Row(y) {
			line.WriteRune(cell.Rune)
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteByte('\n')
	}
	return b.String()
}
