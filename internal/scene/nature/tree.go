// Package nature holds what txoko's scenes share: a tree that grows tip by
// tip, the night sky, the seasons and their weather.
package nature

import (
	"math"
	"math/rand/v2"
	"slices"
	"sync"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// Mark is one character the tree lays down while it grows. tick records
// when: rendering at a given growth shows every mark up to some tick, so the
// tree appears in the same order it was simulated — trunk first, then limbs
// reaching out, then leaves filling in.
type Mark struct {
	X, Y int
	R    rune
	Ink  canvas.Ink
	Tick int
	Leaf bool
}

// Tree is the full simulation result for one seed and canvas size.
type Tree struct {
	Marks []Mark // in growth order: revealing a prefix shows a younger tree
	BaseX int    // where the trunk leaves the ground
	BaseY int    // lowest row the tree may use
	w     int
	used  []bool // cells the fully grown tree occupies, row by row

	// The fully grown crown's bounding box.
	Top, Left, Right int
	// skyline[x] is the highest row the grown tree reaches around column x
	// (baseY+1 where there's no tree), so the sky can follow its silhouette.
	skyline []int
}

// UnderCanopy reports whether (x, y) is at or below the tree's silhouette,
// widened by margin rows: the part of the picture the sky stays out of.
func (t Tree) UnderCanopy(x, y, margin int) bool {
	if x < 0 || x >= len(t.skyline) {
		return false
	}
	return y >= t.skyline[x]-margin && y <= t.BaseY
}

// Floor is the lowest row the tree uses: the sky ends there.
func (t Tree) Floor() int { return t.BaseY }

// WillGrow reports whether the fully grown tree has something at (x, y).
func (t Tree) WillGrow(x, y int) bool {
	if x < 0 || y < 0 || x >= t.w || y*t.w+x >= len(t.used) {
		return false
	}
	return t.used[y*t.w+x]
}

type kind uint8

const (
	trunk kind = iota
	limb       // a big side branch
	twig       // a small branch that ends in leaves
)

// shoot is a growing tip. Each tick it moves one step, leaves a mark behind
// and may sprout new shoots. When its life runs out it bursts into leaves.
type shoot struct {
	x, y      int
	kind      kind
	side      int // -1 grows left, +1 grows right
	life, age int
	lean      int // trunk only: slowly drifting sideways tendency
	lastSpawn int
}

var leafGlyphs = []rune("&&&%%@@#*")

// grower holds the simulation state.
type grower struct {
	rng      *rand.Rand
	w, h     int
	t        Tree
	occupied []bool
	leafR    int // leaf cluster radius, scaled to the canvas
	nextSide int
	limbs    int
	twigs    int
}

// groundRows are the rows below the tree left for whatever it stands on (a
// pot, an island): the trunk starts just above them.
const groundRows = 2

// simulate grows a complete tree for the given seed and canvas size.
func simulate(seed uint64, w, h int) Tree {
	g := &grower{
		rng:      rand.New(rand.NewPCG(seed, uint64(w)<<32|uint64(h)^0xb0b5a1)),
		w:        w,
		h:        h,
		occupied: make([]bool, w*h),
	}
	g.t.BaseX = w / 2
	g.t.BaseY = h - groundRows - 1

	areaH := g.t.BaseY + 1 // rows available for the tree
	if w < 9 || areaH < 3 {
		g.sprout()
		g.finish()
		return g.t
	}
	// In a tall, narrow pane the tree would turn into a pole. Terminal cells
	// are about twice as tall as wide, so a crown ~w cells wide looks right
	// on a tree about w/2 rows tall: cap the height the tree plans for.
	areaH = min(areaH, max(3, w*55/100))
	// Leave the top of the picture to the sky, even in wide, short panes.
	if areaH >= 8 {
		areaH -= max(2, areaH/5)
	}
	g.leafR = min(max(areaH/8, 1), 3)
	g.nextSide = []int{-1, 1}[g.rng.IntN(2)]

	trunkLife := max(2, areaH*45/100+g.rng.IntN(max(1, areaH/5)))
	active := []*shoot{{x: g.t.BaseX, y: g.t.BaseY + 1, kind: trunk, life: trunkLife}}

	for tick := 1; len(active) > 0 && tick < 5000; tick++ {
		var next []*shoot
		for _, s := range active {
			if s.life <= 0 {
				g.cluster(s.x, s.y, tick)
				continue
			}
			next = append(next, g.step(s, tick, trunkLife)...)
		}
		active = next
	}
	g.finish()
	return g.t
}

// finish puts marks in growth order. Marks from the same tick keep the
// order they were laid down in, so the reveal is stable.
func (g *grower) finish() {
	g.t.w, g.t.used = g.w, g.occupied
	g.t.Top, g.t.Left, g.t.Right = g.t.BaseY+1, g.w, -1
	g.t.skyline = make([]int, g.w)
	for x := range g.t.skyline {
		g.t.skyline[x] = g.t.BaseY + 1
	}
	for _, m := range g.t.Marks {
		g.t.Top, g.t.Left, g.t.Right = min(g.t.Top, m.Y), min(g.t.Left, m.X), max(g.t.Right, m.X)
		for x := max(0, m.X-1); x <= min(g.w-1, m.X+1); x++ {
			g.t.skyline[x] = min(g.t.skyline[x], m.Y)
		}
	}
	slices.SortStableFunc(g.t.Marks, func(a, b Mark) int { return a.Tick - b.Tick })
}

// sprout is the whole "tree" when there's barely any room.
func (g *grower) sprout() {
	x, y := g.t.BaseX, g.t.BaseY
	g.put(x, y, '|', canvas.InkWood, 1)
	g.putLeaf(x, y-1, 2)
	g.putLeaf(x-1, y-1, 3)
	g.putLeaf(x+1, y-1, 3)
}

// step advances one shoot and returns it (if still growing) plus any new
// shoots it sprouted.
func (g *grower) step(s *shoot, tick, trunkLife int) []*shoot {
	dx, dy := g.direction(s)
	nx, ny := s.x+dx, s.y+dy
	switch {
	case ny > g.t.BaseY:
		ny, dy = g.t.BaseY, 0
	case ny < 0:
		ny, dy = 0, 0
	}
	if nx < 1 || nx > g.w-2 {
		s.life = 0 // hit the edge: finish here with leaves
		return []*shoot{s}
	}

	ink := canvas.InkWood
	if s.kind == twig || (s.kind == limb && g.rng.Float64() < 0.25) {
		ink = canvas.InkWoodDim
	}
	if dy == 0 && abs(dx) == 2 { // keep long horizontal strides connected
		g.put(s.x+dx/2, ny, '_', ink, tick)
	}
	glyph := g.glyph(s.kind, dx, dy)
	g.put(nx, ny, glyph, ink, tick)
	// The lower trunk is two strokes wide, so it reads as a thick trunk.
	if s.kind == trunk && s.age < trunkLife*6/10 && dy != 0 {
		g.put(nx+1, ny, glyph, canvas.InkWoodDim, tick)
	}
	s.x, s.y = nx, ny

	out := []*shoot{s}
	switch s.kind {
	case trunk:
		if s.age >= trunkLife/3 && s.age-s.lastSpawn >= 2 && g.rng.Float64() < 0.55 && g.limbs < 4+g.w/12 {
			// Higher limbs are shorter, giving the crown its rounded shape.
			reach := max(3, g.w/6+g.rng.IntN(max(1, g.w/8)))
			reach = reach * (s.life + trunkLife) / (2 * trunkLife)
			out = append(out, &shoot{x: s.x, y: s.y, kind: limb, side: g.side(), life: max(2, reach)})
			g.limbs++
			s.lastSpawn = s.age
		}
		if s.life == 1 { // the crown: a few twigs fan out from the top
			for _, side := range []int{-1, 0, 1} {
				out = append(out, g.newTwig(s.x, s.y, side))
			}
		}
	case limb:
		if g.rng.Float64() < 0.22 && g.twigs < 40 {
			out = append(out, g.newTwig(s.x, s.y, s.side))
		}
	}
	// Tips about to finish already carry a few leaves.
	if s.kind != trunk && s.life <= 2 && g.rng.Float64() < 0.5 {
		g.putLeaf(s.x+g.rng.IntN(3)-1, s.y-1, tick+1)
	}

	s.life--
	s.age++
	return out
}

func (g *grower) newTwig(x, y, side int) *shoot {
	g.twigs++
	return &shoot{x: x, y: y, kind: twig, side: side, life: 2 + g.rng.IntN(3+g.leafR)}
}

// side alternates limbs left and right, with a little randomness.
func (g *grower) side() int {
	if g.rng.Float64() < 0.75 {
		g.nextSide = -g.nextSide
	}
	return g.nextSide
}

// direction picks the next step for a shoot. Each kind has its own habit:
// the trunk climbs with a slow lean, limbs reach sideways, twigs wander up.
func (g *grower) direction(s *shoot) (dx, dy int) {
	r := g.rng.Float64
	switch s.kind {
	case trunk:
		if s.age < 2 {
			return 0, -1
		}
		// The lean flips now and then, giving the classic S-shaped trunk.
		if s.lean == 0 {
			s.lean = []int{-1, 1}[g.rng.IntN(2)]
		}
		if r() < 0.2 {
			s.lean = -s.lean
		}
		if r() < 0.5 {
			dx = s.lean
		}
		dy = -1
		if r() > 0.85 {
			dy = 0
		}
		if dx == 0 && dy == 0 {
			dx = []int{-1, 1}[g.rng.IntN(2)]
		}
	case limb:
		dx = s.side
		if r() < 0.3 {
			dx *= 2
		}
		switch v := r(); {
		case v < 0.4:
			dy = -1
		case v > 0.95:
			dy = 1
		}
	default: // twig
		dx = g.rng.IntN(3) - 1
		if r() < 0.4 {
			dx = s.side
		}
		if r() < 0.55 || dx == 0 {
			dy = -1
		}
	}
	return dx, dy
}

func (g *grower) glyph(k kind, dx, dy int) rune {
	switch {
	case dy < 0 && dx == 0:
		return '|'
	case dy < 0 && dx < 0, dy > 0 && dx > 0:
		return '\\'
	case dy < 0, dy > 0 && dx < 0:
		return '/'
	case dy > 0:
		return '|'
	}
	// Moving sideways.
	switch v := g.rng.Float64(); {
	case k == trunk && v < 0.5, v < 0.15:
		return '~'
	case v < 0.25:
		return '='
	}
	return '_'
}

// cluster bursts a shoot's end into a rounded clump of leaves, wider than
// tall because terminal cells are about twice as tall as they are wide.
// Leaves further from the center appear later and more sparsely.
func (g *grower) cluster(cx, cy, tick int) {
	r := g.leafR + g.rng.IntN(2)
	for dy := -r; dy <= (r+1)/2; dy++ {
		for dx := -2 * r; dx <= 2*r; dx++ {
			d := float64(dx*dx)/4 + float64(dy*dy)
			f := d / float64(r*r)
			if f > 1 || g.rng.Float64() > 0.85-0.6*f {
				continue
			}
			g.putLeaf(cx+dx, cy+dy, tick+1+int(math.Sqrt(d)))
		}
	}
}

func (g *grower) putLeaf(x, y, tick int) {
	ink := canvas.InkLeaf
	switch v := g.rng.Float64(); {
	case v < 0.2:
		ink = canvas.InkLeafDim
	case v < 0.5:
		ink = canvas.InkLeafDark
	}
	if g.put(x, y, leafGlyphs[g.rng.IntN(len(leafGlyphs))], ink, tick) {
		g.t.Marks[len(g.t.Marks)-1].Leaf = true
	}
}

// put records a mark if the cell is inside the tree area and still free.
func (g *grower) put(x, y int, r rune, ink canvas.Ink, tick int) bool {
	if x < 0 || x >= g.w || y < 0 || y > g.t.BaseY {
		return false
	}
	i := y*g.w + x
	if g.occupied[i] {
		return false
	}
	g.occupied[i] = true
	g.t.Marks = append(g.t.Marks, Mark{X: x, Y: y, R: r, Ink: ink, Tick: tick})
	return true
}

// The simulation only depends on seed and size, and a scene redraws the
// same tree many times per second, so the last result is cached.
var cache struct {
	sync.Mutex
	seed uint64
	w, h int
	tree *Tree
}

// Grow returns the fully grown tree for a seed on a w×h canvas. The trunk
// starts at the bottom center, groundRows above the last row, and the tree
// plans its size to look right in that space. Results are cached.
func Grow(seed uint64, w, h int) Tree {
	cache.Lock()
	defer cache.Unlock()
	if cache.tree == nil || cache.seed != seed || cache.w != w || cache.h != h {
		t := simulate(seed, w, h)
		cache.seed, cache.w, cache.h, cache.tree = seed, w, h, &t
	}
	return *cache.tree
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
