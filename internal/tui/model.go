// Package tui is txoko's interactive terminal UI, built on Bubble Tea.
//
// Bubble Tea follows the Elm architecture: a Model holds all state, Update
// turns messages (key presses, timer ticks, resizes) into a new model, and
// View renders the model as a string. Nothing else mutates state, which makes
// every behavior easy to unit test by feeding messages to Update.
package tui

import (
	"fmt"
	"hash/fnv"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene/garden"
)

// Defaults for Options.
const (
	DefaultFPS    = 10
	DefaultGrowth = 0.4
	growthStep    = 0.1
)

// Minimum terminal size for the framed view.
const (
	minWidth  = 24
	minHeight = 9
)

// Options configures a new Model. Zero values pick sensible defaults.
type Options struct {
	Seed   uint64           // garden layout; 0 derives one from the hostname
	Growth float64          // starting progress, 0..1
	FPS    int              // animation frames per second
	Now    func() time.Time // clock; injected so tests control day and night
	Theme  *Theme           // nil uses DefaultTheme
}

// skyMode decides whether the sky shows day or night.
type skyMode uint8

const (
	skyAuto skyMode = iota // follow the clock
	skyDay
	skyNight
)

// Model is the TUI state.
type Model struct {
	seed   uint64
	growth float64
	frame  int
	fps    int
	sky    skyMode
	paused bool
	tickID int // identifies the live tick loop; see tickMsg

	width, height int

	now    func() time.Time
	theme  Theme
	canvas *canvas.Canvas
}

// New builds a model from opts.
func New(opts Options) Model {
	m := Model{
		seed:   opts.Seed,
		growth: clamp01(opts.Growth),
		fps:    opts.FPS,
		now:    opts.Now,
		theme:  DefaultTheme(),
		canvas: canvas.New(0, 0),
	}
	if m.seed == 0 {
		m.seed = DefaultSeed()
	}
	if m.fps <= 0 {
		m.fps = DefaultFPS
	}
	if m.now == nil {
		m.now = time.Now
	}
	if opts.Theme != nil {
		m.theme = *opts.Theme
	}
	return m
}

// DefaultSeed derives a stable seed from the machine's hostname, so your
// garden looks the same every time you open it — it's yours.
func DefaultSeed() uint64 {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "txoko"
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(host))
	return h.Sum64() | 1 // never 0, which means "derive one"
}

// tickMsg advances the animation. It carries the ID of the loop that sent
// it: pausing stops the loop and resuming starts a new one, so a stale tick
// still in flight is ignored instead of running two loops at double speed.
type tickMsg struct{ id int }

func (m Model) tick() tea.Cmd {
	id := m.tickID
	return tea.Tick(time.Second/time.Duration(m.fps), func(time.Time) tea.Msg {
		return tickMsg{id: id}
	})
}

// Init starts the animation loop.
func (m Model) Init() tea.Cmd { return m.tick() }

// Update handles a message and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.id != m.tickID || m.paused {
			return m, nil
		}
		m.frame++
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// The scene fills the frame: minus 2 border columns, and minus the
		// top border, separator, footer and bottom border rows.
		m.canvas = canvas.New(max(m.width-2, 0), max(m.height-4, 0))
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "+", "=", "up", "k":
		m.growth = clamp01(m.growth + growthStep)
	case "-", "_", "down", "j":
		m.growth = clamp01(m.growth - growthStep)
	case "n":
		m.sky = (m.sky + 1) % 3
	case "r":
		m.seed = nextSeed(m.seed)
	case "space", "p":
		m.paused = !m.paused
		if !m.paused {
			m.tickID++
			return m, m.tick()
		}
	}
	return m, nil
}

func (m Model) night() bool {
	switch m.sky {
	case skyDay:
		return false
	case skyNight:
		return true
	default:
		h := m.now().Hour()
		return h < 7 || h >= 20
	}
}

// View renders the current state.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "txoko"
	return v
}

func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return "" // waiting for the first WindowSizeMsg
	}
	if m.width < minWidth || m.height < minHeight {
		msg := m.theme.Muted.Render("txoko needs a bit more room 🌱")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	inner := m.canvas.Width()
	night := m.night()
	garden.Draw(m.canvas, m.seed, garden.State{Growth: m.growth, Frame: m.frame, Night: night})

	return frame(inner, m.title(night), renderCanvas(m.canvas, m.theme), m.footer(inner), m.theme)
}

func (m Model) title(night bool) []span {
	t := m.theme
	sky := "day"
	if night {
		sky = "night"
	}
	title := []span{
		{"txoko", t.Title},
		{" · garden · " + sky, t.Muted},
	}
	if m.paused {
		title = append(title, span{" · paused", t.Accent})
	}
	return title
}

func (m Model) footer(inner int) string {
	t := m.theme
	filled, empty := progressBar(m.growth, 10)
	bar := []span{
		{filled, t.Ink[canvas.InkLeaf]},
		{empty, t.Muted},
		{fmt.Sprintf(" %3.0f%%", m.growth*100), t.Accent},
	}
	full := append([]span{{"growth ", t.Muted}}, bar...)
	full = append(full, span{"  demo", t.Muted})

	hints := []span{
		{"+/-", t.Accent},
		{" grow  ", t.Muted},
		{"n", t.Accent},
		{" sky  ", t.Muted},
		{"r", t.Accent},
		{" reseed  ", t.Muted},
		{"space", t.Accent},
		{" pause  ", t.Muted},
		{"q", t.Accent},
		{" quit", t.Muted},
	}
	shortHints := []span{{"+/-", t.Accent}, {" grow  ", t.Muted}, {"q", t.Accent}, {" quit", t.Muted}}

	// From richest to most compact; the first that fits wins.
	return footerLine(inner,
		layout{full, hints},
		layout{full, shortHints},
		layout{full, nil},
		layout{bar, nil},
	)
}

// nextSeed derives a new seed deterministically (splitmix64), so reseeding
// is reproducible in tests.
func nextSeed(s uint64) uint64 {
	s += 0x9e3779b97f4a7c15
	z := s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return (z ^ (z >> 31)) | 1
}

func clamp01(v float64) float64 {
	switch {
	case v != v: // NaN
		return 0
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}
