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
	"math/rand/v2"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// Defaults for Options.
const (
	DefaultFPS     = 10
	DefaultGrowth  = 0.4
	DefaultRefresh = 3 * time.Second
	growthStep     = 0.1
	// easing is the share of the remaining distance the scene grows each
	// frame, so new activity makes it grow smoothly instead of jumping.
	easing = 0.1
)

// Minimum terminal size for the framed view.
const (
	minWidth  = 24
	minHeight = 9
)

// Options configures a new Model. Zero values pick sensible defaults.
type Options struct {
	Scene  string  // scene name; "" picks the default (see SceneNames)
	Season string  // "spring", "summer", "autumn" or "winter"; "" follows the calendar
	Seed   uint64  // scene layout; 0 derives one from the hostname
	Growth float64 // manual mode: starting progress, 0..1
	FPS    int     // animation frames per second

	// Source reads the real activity summary (txoko --work). When nil the
	// UI runs in manual mode, where +/- set the growth by hand.
	Source  func() (progress.Summary, error)
	Refresh time.Duration // how often Source is read again

	// Focus starts a pomodoro session of this length: the scene grows from
	// a seed to full bloom as the minutes pass. Break adds a rest after it.
	Focus, Break time.Duration
	// OnFocusDone runs (outside the UI loop) when a session completes, to
	// record it and notify. OnBreakDone runs when the break is over.
	OnFocusDone func(length time.Duration) error
	OnBreakDone func()

	Now   func() time.Time // clock; injected so tests control day, night and timers
	Theme *Theme           // nil uses DefaultTheme

	// MoonPhase fixes the moon's phase (0 new, 0.5 full); nil picks a random
	// one, and r rolls a new one along with a new tree.
	MoonPhase *float64
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
	scene  int // index into scenes
	seed   uint64
	growth float64
	frame  int
	fps    int
	sky    skyMode
	// season is fixed when seasonFixed is set; otherwise it follows the
	// calendar through now().
	season      scene.Season
	seasonFixed bool
	paused      bool
	tickID      int // identifies the live tick loop; see tickMsg

	source  func() (progress.Summary, error)
	refresh time.Duration
	target  float64 // live mode: the growth the scene is easing toward
	summary progress.Summary
	loaded  bool
	loadErr error

	moonPhase float64

	pomo        pomodoro
	onFocusDone func(time.Duration) error
	onBreakDone func()

	width, height int

	now    func() time.Time
	theme  Theme
	canvas *canvas.Canvas
}

// New builds a model from opts.
func New(opts Options) Model {
	m := Model{
		seed:    opts.Seed,
		growth:  clamp01(opts.Growth),
		fps:     opts.FPS,
		now:     opts.Now,
		theme:   DefaultTheme(),
		canvas:  canvas.New(0, 0),
		source:  opts.Source,
		refresh: opts.Refresh,

		onFocusDone: opts.OnFocusDone,
		onBreakDone: opts.OnBreakDone,
	}
	if m.live() {
		m.growth = 0 // the scene grows in from a seed when it opens
	}
	if m.refresh <= 0 {
		m.refresh = DefaultRefresh
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
	m.scene, _ = sceneIndex(opts.Scene) // unknown names fall back to the default
	if s, ok := scene.ParseSeason(opts.Season); ok {
		m.season, m.seasonFixed = s, true
	}
	m.moonPhase = randomMoon()
	if opts.MoonPhase != nil {
		m.moonPhase = *opts.MoonPhase
	}
	if opts.Focus > 0 {
		m.pomo = pomodoro{phase: phaseFocus, focus: opts.Focus, brk: opts.Break, left: opts.Focus, last: m.now()}
		m.growth = 0
	}
	return m
}

// DefaultSeed derives a stable seed from the machine's hostname, so your
// scene looks the same every time you open it — it's yours.
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

// summaryMsg carries a fresh read of the activity data.
type summaryMsg struct {
	summary progress.Summary
	err     error
}

func (m Model) live() bool { return m.source != nil }

// load reads the source. Commands run outside the update loop, so a slow
// disk never freezes the animation.
func (m Model) load() tea.Msg {
	s, err := m.source()
	return summaryMsg{summary: s, err: err}
}

func (m Model) scheduleLoad() tea.Cmd {
	return tea.Tick(m.refresh, func(time.Time) tea.Msg { return m.load() })
}

// Init starts the animation loop and, in live mode, the first data read.
func (m Model) Init() tea.Cmd {
	if m.live() {
		return tea.Batch(m.tick(), m.load)
	}
	return m.tick()
}

// Update handles a message and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.id != m.tickID || m.paused {
			return m, nil
		}
		m.frame++
		var phaseCmd tea.Cmd
		if m.pomodoroOn() {
			m, phaseCmd = m.advance()
			m.target = m.pomoTarget()
		}
		if m.live() || m.pomodoroOn() {
			m.growth += (m.target - m.growth) * easing
			if abs(m.target-m.growth) < 0.001 {
				m.growth = m.target
			}
		}
		if phaseCmd != nil {
			return m, tea.Batch(m.tick(), phaseCmd)
		}
		return m, m.tick()

	case focusSavedMsg:
		m.pomo.saveErr = msg.err
		return m, nil

	case summaryMsg:
		m.loaded, m.loadErr = true, msg.err
		if msg.err == nil {
			m.summary = msg.summary
			m.target = clamp01(msg.summary.Growth)
		}
		return m, m.scheduleLoad()

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
	if m.pomodoroOn() {
		next, cmd, handled := m.handlePomodoroKey(msg.String())
		if handled {
			return next, cmd
		}
		m = next
	}
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "+", "=", "up", "k":
		if !m.live() {
			m.growth = clamp01(m.growth + growthStep)
		}
	case "-", "_", "down", "j":
		if !m.live() {
			m.growth = clamp01(m.growth - growthStep)
		}
	case "e":
		m = m.nextSeason()
	case "n":
		// One press flips what you see: day becomes night and back.
		if m.night() {
			m.sky = skyDay
		} else {
			m.sky = skyNight
		}
	case "s":
		m.scene = (m.scene + 1) % len(scenes)
	case "r":
		m.seed = nextSeed(m.seed)
		m.moonPhase = randomMoon()
	case "space", "p":
		m.paused = !m.paused
		if !m.paused {
			m.tickID++
			m.pomo.last = m.now() // paused time doesn't count toward the session
			return m, m.tick()
		}
	}
	return m, nil
}

// currentSeason is the fixed season, or the calendar's.
func (m Model) currentSeason() scene.Season {
	if m.seasonFixed {
		return m.season
	}
	return scene.SeasonOf(m.now())
}

// nextSeason cycles calendar → spring → summer → autumn → winter → calendar.
func (m Model) nextSeason() Model {
	if !m.seasonFixed {
		m.season, m.seasonFixed = scene.Seasons[0], true
		return m
	}
	for i, s := range scene.Seasons {
		if s == m.season {
			if i == len(scene.Seasons)-1 {
				m.seasonFixed = false
			} else {
				m.season = scene.Seasons[i+1]
			}
			break
		}
	}
	return m
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

	inner := m.width - 2
	footer := m.footer(inner)
	// The scene fills what the borders, the title and the footer leave.
	m.canvas.Resize(inner, max(1, m.height-3-len(footer)))
	night := m.drawScene()
	return frame(inner, m.title(night, inner), renderCanvas(m.canvas, m.theme), footer, m.theme)
}

// drawScene paints the current scene onto the canvas and reports whether
// it is night.
func (m Model) drawScene() bool {
	night := m.night()
	scenes[m.scene].Draw(m.canvas, m.seed, scene.State{
		Growth: m.growth, Frame: m.frame, Night: night, Season: m.currentSeason(),
		MoonPhase: m.moonPhase,
	})
	return night
}

// title builds the frame's title, dropping the least important parts until
// it fits in inner cells: scene name first, then the season, then the sky.
// What's happening right now (focus, break, paused) is kept the longest.
func (m Model) title(night bool, inner int) []span {
	t := m.theme
	sky := "day"
	if night {
		sky = "night"
	}
	var status []span
	switch m.pomo.phase {
	case phaseFocus:
		status = append(status, span{" · focus", t.Accent})
	case phaseBreak:
		status = append(status, span{" · break", t.Accent})
	}
	if m.paused {
		status = append(status, span{" · paused", t.Accent})
	}

	scene, season := scenes[m.scene].Name(), m.currentSeason().String()
	for _, details := range []string{
		" · " + scene + " · " + season + " · " + sky,
		" · " + season + " · " + sky,
		" · " + sky,
		"",
	} {
		title := append([]span{{"txoko", t.Title}, {details, t.Muted}}, status...)
		if spansWidth(title)+3 <= inner {
			return title
		}
	}
	return []span{{"txoko", t.Title}}
}

// hint is one entry of the key cheat sheet: the key and what it does.
func (m Model) hint(key, label string) []span {
	return []span{{key, m.theme.Accent}, {" " + label, m.theme.Muted}}
}

func (m Model) footer(inner int) []string {
	status, compact, hints := m.footerParts()
	return footerBlock(inner, status, compact, hints)
}

// footerParts returns the footer's pieces: the status, a shorter status for
// narrow spaces, and the key hints.
func (m Model) footerParts() (status, compact []span, hints [][]span) {
	if m.pomodoroOn() {
		return m.pomodoroParts()
	}
	t := m.theme
	filled, empty := progressBar(m.growth, 10)
	bar := []span{
		{filled, t.Ink[canvas.InkLeaf]},
		{empty, t.Muted},
		{fmt.Sprintf(" %3.0f%%", m.growth*100), t.Accent},
	}
	status = append([]span{{"growth ", t.Muted}}, bar...)
	status = append(status, m.status()...)

	if !m.live() {
		hints = append(hints, m.hint("+/-", "grow"))
	}
	hints = append(hints,
		m.hint("s", "scene"),
		m.hint("e", "season"),
		m.hint("n", "night"),
		m.hint("r", "reseed"),
		m.hint("space", "pause"),
		m.hint("q", "quit"),
	)
	return status, bar, hints
}

// status is the part of the footer that describes where growth comes from.
func (m Model) status() []span {
	t := m.theme
	switch {
	case !m.live():
		return []span{{"  manual", t.Muted}}
	case m.loadErr != nil:
		return []span{{"  can't read activity", t.Accent}}
	case !m.loaded:
		return nil
	case m.summary.Recent == 0:
		return []span{{"  no commits yet", t.Muted}}
	}
	return []span{
		{"  today ", t.Muted},
		{fmt.Sprint(m.summary.Today), t.Accent},
		{" · streak ", t.Muted},
		{fmt.Sprintf("%dd", m.summary.Streak), t.Accent},
	}
}

// randomMoon picks a moon phase that's nice to look at: never so close to
// new moon that it all but disappears.
func randomMoon() float64 { return 0.15 + rand.Float64()*0.7 }

// nextSeed derives a new seed deterministically (splitmix64), so reseeding
// is reproducible in tests.
func nextSeed(s uint64) uint64 {
	s += 0x9e3779b97f4a7c15
	z := s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return (z ^ (z >> 31)) | 1
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
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
