// Package arch draws a forgotten arch: an old stone wall with an arch in
// it, ivy and moss slowly taking over the stone and, through the arch, a
// meadow where a young tree grows. The tree, the sky and the seasons come
// from package nature, so it shares the bonsai's look.
package arch

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
)

// Growth thresholds for the meadow's own details.
const (
	flowersAt     = 0.35 // flowers open in the meadow
	butterfliesAt = 0.6  // butterflies come out on warm days
)

// Scene exposes the arch through the scene.Scene interface.
type Scene struct{}

// Name implements scene.Scene.
func (Scene) Name() string { return "arch" }

// Draw implements scene.Scene.
func (Scene) Draw(c *canvas.Canvas, seed uint64, s scene.State) { Draw(c, seed, s) }

// layout is where everything goes for a given canvas size.
type layout struct {
	w, h    int
	cx      int   // the arch's center column
	aw      int   // half width of the opening
	rv      int   // rows the curve rises above the springline
	ys      int   // the springline, where the curve starts
	archTop int   // the opening's highest row
	meadow  int   // the meadow seen through the arch; the wall's last row
	ground  int   // the grass in front of the wall
	top     []int // per column: the wall's first row
	x0, x1  int   // the wall's ends
	tree    nature.Tree
	treeDX  int  // the tree's offset on the canvas
	plain   bool // no room for a wall: just the tree on the grass
	ivy     map[point]ivyCell
}

type point struct{ x, y int }

func newLayout(seed uint64, w, h int) layout {
	l := layout{w: w, h: h, cx: w / 2, meadow: h - 2, ground: h - 1}
	if w < 16 || h < 8 {
		l.plain = true
		l.tree = nature.Grow(seed, w, h)
		return l
	}
	l.aw = min(max(w/8, 3), 10)
	l.rv = max(2, int(math.Round(float64(l.aw)*0.55)))
	// The opening is shaped like a real doorway, about twice as tall as it
	// is wide (cells are twice as tall as wide), never a slit: in a tall
	// pane the wall stays low and leaves the rest to the sky.
	open := min(max(int(math.Round(float64(h)*0.55)), l.rv+3), 2*l.aw, h-3)
	l.archTop = l.meadow - open + 1
	l.ys = l.archTop + l.rv
	l.top = l.crumble(max(0, l.archTop-2))

	// A young tree grows in the meadow, right behind the arch. It plans its
	// size from the opening, so it stays young and leaves the arch's crown
	// to the moon.
	tw := min(w, 2*l.aw+8)
	l.tree = nature.Grow(seed, tw, h)
	l.treeDX = l.cx - tw/2
	l.ivy = l.grow(seed)
	return l
}

// crumble gives the wall a ruined top. What's left of it stands in the
// middle of the meadow with sky on both sides; stone by stone it is a row
// or two lower here and there, and it falls away towards both ends. Over
// the arch it keeps a course of stone above the keystone.
func (l *layout) crumble(wallTop int) []int {
	hw := min(max(int(math.Round(float64(l.w)*0.36)), l.aw+6), l.w/2)
	l.x0, l.x1 = l.cx-hw, min(l.cx+hw, l.w-1)
	top := make([]int, l.w)
	for x := range top {
		top[x] = l.meadow + 1 // no wall
	}
	wallH := l.meadow - wallTop
	edge := max(2, hw/2)
	for x, stone := l.x0, 0; x <= l.x1; stone++ {
		n := 3 + int(nature.Noise(stone, 0, 71)*4)
		mid := x + n/2
		drop := int(nature.Noise(stone, 1, 73) * 2.99)
		if d := min(mid-l.x0, l.x1-mid); d < edge {
			drop += int(math.Round(float64(edge-d) / float64(edge) * float64(wallH) * 0.7))
		}
		for i := x; i <= min(x+n-1, l.x1); i++ {
			top[i] = min(wallTop+drop, l.meadow)
			if abs(i-l.cx) <= l.aw+1 {
				top[i] = min(top[i], l.archTop-2)
			}
		}
		x += n
	}
	return top
}

// arcY is the opening's first row in column x.
func (l layout) arcY(x int) int {
	dx := float64(x-l.cx) / float64(l.aw)
	return l.ys - int(math.Round(float64(l.rv)*math.Sqrt(max(0, 1-dx*dx))))
}

// open reports whether (x, y) is inside the arch, looking into the meadow.
func (l layout) open(x, y int) bool {
	return !l.plain && abs(x-l.cx) < l.aw && y >= l.arcY(x) && y <= l.meadow
}

// wall reports whether (x, y) is stone.
func (l layout) wall(x, y int) bool {
	return !l.plain && x >= l.x0 && x <= l.x1 && y >= l.top[x] && y <= l.meadow && !l.open(x, y)
}

// ring reports whether (x, y) is one of the arch's own stones, the thin
// outline right around the opening.
func (l layout) ring(x, y int) bool {
	return l.wall(x, y) && (l.open(x, y+1) || l.open(x-1, y) || l.open(x+1, y))
}

// UnderCanopy keeps the sky off the wall, the tree and the ground.
func (l layout) UnderCanopy(x, y, margin int) bool {
	if y >= l.meadow-margin || l.tree.UnderCanopy(x-l.treeDX, y, margin) {
		return true
	}
	for k := 0; k <= margin; k++ {
		if l.wall(x, y+k) {
			return true
		}
	}
	return false
}

// Floor is the lowest row of the sky, just above the meadow.
func (l layout) Floor() int { return l.meadow - 1 }

// MoonFrame puts the moon in the arch, under its keystone, as big as the
// opening allows. It hangs behind the young tree, peeking through its
// crown.
func (l layout) MoonFrame() (cx, top, maxR int) {
	if l.plain {
		return 0, 0, 0
	}
	top = l.archTop + 1
	for r := 6; r >= 2; r-- {
		fits := true
		for y := top; y < top+r && fits; y++ {
			for x := l.cx - r - 1; x <= l.cx+r; x++ {
				if !l.open(x, y) {
					fits = false
					break
				}
			}
		}
		if fits {
			return l.cx, top, r
		}
	}
	return 0, 0, 0
}

// Draw paints the arch onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, s scene.State) {
	l, leaves, g, ok := drawStatic(c, seed, s)
	if !ok {
		return
	}
	if s.Night {
		nature.DimForNight(c)
		nature.DrawSky(c, seed, l, s.MoonPhase, s.Frame)
		nature.DrawFireflies(c, l.meadow, leaves, g, s.Frame, s.Season)
	} else {
		if !l.plain {
			nature.DrawClouds(c, seed, l, []int{1, l.top[l.w/4] / 2, l.archTop + 2}, s.Frame)
		}
		if g >= butterfliesAt && (s.Season == scene.Summer || s.Season == scene.Spring) {
			drawButterflies(c, l, s.Frame)
		}
	}
	nature.DrawWeather(c, seed, l.ground, leaves, g, s.Frame, s.Season)
}

// drawStatic paints everything that doesn't move: the tree behind the
// wall, the wall and its arch, the ivy and moss on it, the meadow and the
// grass, all dressed for the season. Growing only ever covers more stone
// with green; nothing that grew ever goes away. It returns the leaves you
// can see, for the weather and the fireflies.
func drawStatic(c *canvas.Canvas, seed uint64, s scene.State) (l layout, leaves []nature.Mark, g float64, ok bool) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w < 6 || h < 4 {
		return layout{}, nil, 0, false
	}
	l = newLayout(seed, w, h)
	g = nature.Clamp01(s.Growth)

	visible, all := nature.Reveal(c, l.tree, g, s.Season, l.treeDX)
	if s.Season == scene.Winter {
		nature.SnowOnBranches(c, l.tree, visible, l.treeDX)
	}
	drawWall(c, l, s.Season)
	drawMoss(c, l, g, s.Season)
	drawIvy(c, l, g, s.Season)
	drawHills(c, l, s.Season)
	drawGrass(c, l, l.meadow, g, s.Season, true)
	drawGrass(c, l, l.ground, g, s.Season, false)

	for _, m := range all {
		if !l.wall(m.X, m.Y) {
			leaves = append(leaves, m)
		}
	}
	return l, leaves, g, true
}

// drawWall lays the stones in courses two rows tall with staggered
// joints, outlines the arch and, in winter, puts snow on top:
//
//	|       |    ____    |
//	|_______|___/    \___|
//	    |       |    |   |
//	____|_______|    |___|
func drawWall(c *canvas.Canvas, l layout, season scene.Season) {
	if l.plain {
		return
	}
	for x := l.x0; x <= l.x1; x++ {
		for y := l.top[x]; y <= l.meadow; y++ {
			switch {
			case !l.wall(x, y):
			case l.ring(x, y):
				c.Set(x, y, l.ringGlyph(x, y), canvas.InkRock)
			case season == scene.Winter && y == l.top[x] && !l.hasIvy(x, y):
				c.Set(x, y, []rune{'_', '.', '_'}[x%3], canvas.InkSnow)
			default:
				if r := stone(x, l.meadow-y); r != ' ' || y == l.top[x] {
					if r == ' ' {
						r = '_' // the top of the wall always shows
					}
					c.Set(x, y, r, canvas.InkRockDim)
				}
			}
		}
	}
}

// stone is the brick pattern at column x, rel rows above the wall's foot:
// the bottom row of each course is mortar, joints are staggered.
func stone(x, rel int) rune {
	course := rel / 2
	if (x+(course%2)*4)%8 == 0 {
		return '|'
	}
	if rel%2 == 0 {
		return '_'
	}
	return ' '
}

// ringGlyph draws the arch's outline: jambs up the sides, then the curve
// stepping in towards the keystone.
func (l layout) ringGlyph(x, y int) rune {
	below, side := l.open(x, y+1), l.open(x-1, y) || l.open(x+1, y)
	switch {
	case below && !side && l.open(x-1, y+1) && l.open(x+1, y+1):
		return '_'
	case below || !side && !below:
		if x < l.cx {
			return '/'
		}
		return '\\'
	}
	return '|'
}

// drawMoss lets moss creep over the stones, mostly low on the wall where
// it's damp. Each stone has its own moment to turn green.
func drawMoss(c *canvas.Canvas, l layout, g float64, season scene.Season) {
	if l.plain {
		return
	}
	ink := canvas.InkGrass
	if season == scene.Winter {
		ink = canvas.InkLeafDim
	}
	for x := l.x0; x <= l.x1; x++ {
		for y := l.top[x] + 1; y <= l.meadow; y++ {
			if !l.wall(x, y) || l.ring(x, y) || l.hasIvy(x, y) {
				continue
			}
			low := float64(y-l.top[x]) / float64(max(1, l.meadow-l.top[x]))
			if nature.Noise(x, y, 81) > 0.03+0.22*low*low {
				continue
			}
			if g > nature.Noise(x, y, 83)*0.95 {
				c.Set(x, y, []rune(",.',\"")[int(nature.Noise(x, y, 85)*5)], ink)
			}
		}
	}
}

// hills is the row of the far hills seen through the arch.
func (l layout) hills() int { return l.meadow - max(1, (l.meadow-l.ys)/2) }

// drawHills draws the far hills behind the tree, as one gentle line.
func drawHills(c *canvas.Canvas, l layout, season scene.Season) {
	if l.plain {
		return
	}
	ink := canvas.InkLeafDim
	switch season {
	case scene.Autumn:
		ink = canvas.InkAutumnGold
	case scene.Winter:
		ink = canvas.InkSnow
	}
	y := l.hills()
	heights := []rune("_.-~'")
	for x := l.cx - l.aw + 1; x < l.cx+l.aw; x++ {
		v := (math.Sin(float64(x)*0.45+float64(l.w%7)) + math.Sin(float64(x)*0.17)) / 4 // -0.5..0.5
		c.SetIfEmpty(x, y, heights[int((v+0.5)*float64(len(heights)-1)+0.5)], ink)
	}
}

// drawGrass grows grass along a row: the meadow around and through the
// wall, or the strip in front of it. Flowers, fallen leaves or snow come with
// the season. Each spot decides once what grows there; growth only decides
// when it shows up.
func drawGrass(c *canvas.Canvas, l layout, y int, g float64, season scene.Season, meadow bool) {
	if meadow && l.plain {
		return
	}
	blades := []rune(",'`.,'\"")
	for x := range l.w {
		appear := nature.Noise(x, y, 41) * 0.9
		glyph, ink := blades[int(nature.Noise(x, y, 43)*float64(len(blades)))], canvas.InkGrass
		switch bloom := nature.Noise(x, y, 47); {
		case season == scene.Winter:
			glyph, ink, appear = []rune{'_', '.', '_'}[x%3], canvas.InkSnow, -1
		case season == scene.Autumn && bloom < 0.3:
			glyph, ink = ',', []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}[x%3]
		case season == scene.Spring && bloom < 0.2:
			glyph, ink, appear = '✿', canvas.InkBlossom, max(appear, flowersAt)
		case season == scene.Summer && bloom < 0.14:
			glyph, ink, appear = '✿', canvas.InkPetalYellow, max(appear, flowersAt)
		case bloom > 0.94:
			glyph, appear = 'ψ', max(appear, 0.3)
		}
		if g > appear {
			c.SetIfEmpty(x, y, glyph, ink)
		}
	}
}

// drawButterflies sends two butterflies fluttering over the meadow and
// along the foot of the wall.
func drawButterflies(c *canvas.Canvas, l layout, frame int) {
	if l.plain {
		return
	}
	for i := range 2 {
		x := l.cx - l.aw - 4 + nature.Triangle(frame/3+i*17, 2*l.aw+8)
		y := l.meadow - 1 - nature.Triangle(frame/5+i*7, max(2, (l.meadow-l.ys)/2))
		glyph := 'v'
		if (frame/2+i)%2 == 0 {
			glyph = '^'
		}
		c.Set(x, y, glyph, canvas.InkButterfly)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
