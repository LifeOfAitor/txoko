package tui

import (
	"fmt"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// phase is where a pomodoro session stands.
type phase uint8

const (
	phaseOff   phase = iota // not a pomodoro: manual or --work mode
	phaseFocus              // the tree grows from seed to bloom
	phaseBreak              // the tree rests in full bloom
	phaseDone               // waiting for enter to start another session
)

// pomodoro is the focus timer. Time is measured with the injected clock, so
// tests can fast-forward a 50-minute session in microseconds.
type pomodoro struct {
	phase      phase
	focus, brk time.Duration
	left       time.Duration // remaining in the current phase
	last       time.Time     // when the timer last advanced
	sessions   int           // focus sessions completed in this run
	quitArmed  bool          // q was pressed once; a second q abandons
	saveErr    error
}

// focusSavedMsg reports whether a finished session was recorded.
type focusSavedMsg struct{ err error }

func (m Model) pomodoroOn() bool { return m.pomo.phase != phaseOff }

func (m Model) timing() bool {
	return m.pomo.phase == phaseFocus || m.pomo.phase == phaseBreak
}

// advance moves the timer forward by the real time since the last tick and
// handles the end of a phase.
func (m Model) advance() (Model, tea.Cmd) {
	p := &m.pomo
	now := m.now()
	elapsed := now.Sub(p.last)
	p.last = now
	if !m.timing() {
		return m, nil
	}
	p.left -= elapsed
	if p.left > 0 {
		return m, nil
	}

	if p.phase == phaseBreak {
		p.phase, p.left = phaseDone, 0
		if done := m.onBreakDone; done != nil {
			return m, func() tea.Msg { done(); return nil }
		}
		return m, nil
	}

	// Focus finished: the tree is in full bloom.
	p.sessions++
	length := p.focus
	if p.brk > 0 {
		p.phase, p.left = phaseBreak, p.brk
	} else {
		p.phase, p.left = phaseDone, 0
	}
	if done := m.onFocusDone; done != nil {
		return m, func() tea.Msg { return focusSavedMsg{err: done(length)} }
	}
	return m, nil
}

// pomoTarget is the growth the scene should show for the current phase:
// rising with focus time, then fully grown.
func (m Model) pomoTarget() float64 {
	if m.pomo.phase == phaseFocus && m.pomo.focus > 0 {
		return clamp01(1 - float64(m.pomo.left)/float64(m.pomo.focus))
	}
	return 1
}

// newSession plants a new tree and starts another focus session.
func (m Model) newSession() Model {
	m.seed = nextSeed(m.seed)
	m.pomo.phase, m.pomo.left, m.pomo.last = phaseFocus, m.pomo.focus, m.now()
	m.pomo.quitArmed, m.pomo.saveErr = false, nil
	m.growth, m.target = 0, 0
	return m
}

// handlePomodoroKey handles the keys that behave differently during a
// session. It reports whether the key was fully handled.
func (m Model) handlePomodoroKey(key string) (Model, tea.Cmd, bool) {
	if key != "q" && key != "esc" {
		m.pomo.quitArmed = false
	}
	switch key {
	case "q", "esc":
		// Quitting mid-session asks twice, so a stray q never throws away
		// forty minutes of focus.
		if m.timing() && !m.pomo.quitArmed {
			m.pomo.quitArmed = true
			return m, nil, true
		}
		return m, tea.Quit, true
	case "enter":
		if m.pomo.phase == phaseDone {
			return m.newSession(), nil, true
		}
		return m, nil, true
	case "+", "=", "up", "k", "-", "_", "down", "j":
		return m, nil, true // the timer decides the growth
	}
	return m, nil, false
}

func (m Model) pomodoroParts() (status, compact []span, hints [][]span) {
	t := m.theme
	p := m.pomo
	quit := m.hint("q", "quit")

	switch {
	case p.quitArmed:
		status = []span{{"press q again to abandon this session", t.Accent}}
		hints = [][]span{m.hint("any key", "keep going"), quit}
	case p.phase == phaseFocus:
		filled, empty := progressBar(m.pomoTarget(), 10)
		status = []span{
			{"focus ", t.Muted},
			{filled, t.Ink[canvas.InkLeaf]},
			{empty, t.Muted},
			{" " + clock(p.left), t.Accent},
		}
		hints = [][]span{m.hint("space", "pause"), m.hint("s", "scene"), m.hint("e", "season"), m.hint("n", "night"), quit}
	case p.phase == phaseBreak:
		status = []span{{"break ", t.Muted}, {clock(p.left), t.Accent}, {"  stretch, breathe", t.Muted}}
		hints = [][]span{m.hint("space", "pause"), m.hint("n", "night"), quit}
	default:
		status = []span{{"✿ session complete", t.Accent}, {fmt.Sprintf("  %s today", plural(p.sessions, "session")), t.Muted}}
		hints = [][]span{m.hint("enter", "plant another"), quit}
	}
	if p.saveErr != nil {
		status = append(status, span{"  (not saved)", t.Accent})
	}
	return status, status, hints
}

// clock formats a duration as mm:ss, rounding up so a session shows 50:00
// when it starts and 00:01 in its last second.
func clock(d time.Duration) string {
	secs := max(0, int(math.Ceil(d.Seconds())))
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
