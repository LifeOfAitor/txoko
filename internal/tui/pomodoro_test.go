package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// fakeClock is a clock the test moves by hand.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time                { return c.t }
func (c *fakeClock) add(d time.Duration)           { c.t = c.t.Add(d) }
func newClock() *fakeClock                         { return &fakeClock{t: time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)} }
func (c *fakeClock) tick(m Model) (Model, tea.Cmd) { return send(m, tickMsg{id: m.tickID}) }

type recorder struct {
	focus  []time.Duration
	breaks int
	err    error
}

func newPomo(clock *fakeClock, focus, brk time.Duration, rec *recorder) Model {
	plain := PlainTheme()
	return New(Options{
		Seed: 42, Theme: &plain, Now: clock.now, Focus: focus, Break: brk, MoonPhase: &testMoon,
		OnFocusDone: func(d time.Duration) error { rec.focus = append(rec.focus, d); return rec.err },
		OnBreakDone: func() { rec.breaks++ },
	})
}

// runCmd executes cmd once and feeds the messages it produces back into m.
// Tick messages are dropped: in a real program they arrive later.
func runCmd(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	return feed(m, cmd())
}

func feed(m Model, msg tea.Msg) Model {
	switch msg := msg.(type) {
	case nil, tickMsg:
	case tea.BatchMsg:
		for _, c := range msg {
			if c != nil {
				m = feed(m, c())
			}
		}
	default:
		m, _ = send(m, msg)
	}
	return m
}

func TestPomodoroGrowsWithTheClock(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, 50*time.Minute, 0, &recorder{})
	if m.growth != 0 || m.pomo.phase != phaseFocus {
		t.Fatalf("a session starts as a seed in focus: %+v", m.pomo)
	}

	clock.add(25 * time.Minute)
	m, _ = clock.tick(m)
	if m.target != 0.5 {
		t.Errorf("half way: target = %v, want 0.5", m.target)
	}
	if got := clock2(m); got != "25:00" {
		t.Errorf("timer shows %s, want 25:00", got)
	}
}

func clock2(m Model) string { return clock(m.pomo.left) }

func TestPomodoroCompletes(t *testing.T) {
	clock := newClock()
	rec := &recorder{}
	m := newPomo(clock, 50*time.Minute, 0, rec)

	clock.add(50*time.Minute + time.Second)
	m, cmd := clock.tick(m)
	m = runCmd(m, cmd)

	if m.pomo.phase != phaseDone || m.pomo.sessions != 1 {
		t.Fatalf("after 50 min: phase %d, sessions %d", m.pomo.phase, m.pomo.sessions)
	}
	if len(rec.focus) != 1 || rec.focus[0] != 50*time.Minute {
		t.Errorf("OnFocusDone calls = %v, want one with 50m", rec.focus)
	}
	if m.target != 1 {
		t.Errorf("a finished session shows full bloom, target %v", m.target)
	}
}

func TestPomodoroBreakThenAnother(t *testing.T) {
	clock := newClock()
	rec := &recorder{}
	m := newPomo(clock, 25*time.Minute, 5*time.Minute, rec)
	seed := m.seed

	clock.add(25 * time.Minute)
	m, cmd := clock.tick(m)
	m = runCmd(m, cmd)
	if m.pomo.phase != phaseBreak || clock2(m) != "05:00" {
		t.Fatalf("expected a 5 minute break, got phase %d left %s", m.pomo.phase, clock2(m))
	}

	clock.add(5 * time.Minute)
	m, cmd = clock.tick(m)
	m = runCmd(m, cmd)
	if m.pomo.phase != phaseDone || rec.breaks != 1 {
		t.Fatalf("break should end: phase %d, OnBreakDone calls %d", m.pomo.phase, rec.breaks)
	}

	m, _ = send(m, key("enter"))
	if m.pomo.phase != phaseFocus || m.growth != 0 || m.seed == seed || clock2(m) != "25:00" {
		t.Errorf("enter should plant a new tree and restart: phase %d growth %v left %s", m.pomo.phase, m.growth, clock2(m))
	}
}

func TestPomodoroPauseStopsTheClock(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, 50*time.Minute, 0, &recorder{})

	clock.add(10 * time.Minute)
	m, _ = clock.tick(m)
	m, _ = send(m, key("space")) // pause
	clock.add(30 * time.Minute)  // a long coffee
	m, _ = send(m, key("space")) // resume
	clock.add(time.Minute)
	m, _ = clock.tick(m)

	if got := clock2(m); got != "39:00" {
		t.Errorf("paused time must not count: %s left, want 39:00", got)
	}
}

func TestPomodoroQuitNeedsConfirmation(t *testing.T) {
	m := newPomo(newClock(), 50*time.Minute, 0, &recorder{})

	m, cmd := send(m, key("q"))
	if cmd != nil || !m.pomo.quitArmed {
		t.Fatal("first q should only ask for confirmation")
	}
	m, _ = send(m, key("x"))
	if m.pomo.quitArmed {
		t.Fatal("any other key should cancel the confirmation")
	}
	m, _ = send(m, key("q"))
	_, cmd = send(m, key("q"))
	if cmd == nil {
		t.Fatal("second q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("second q should quit")
	}

	// ctrl+c always quits immediately.
	_, cmd = send(newPomo(newClock(), time.Minute, 0, &recorder{}), key("ctrl+c"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c should quit at once")
	}
}

func TestPomodoroQuitsAtOnceWhenDone(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, time.Minute, 0, &recorder{})
	clock.add(time.Minute)
	m, _ = clock.tick(m)
	_, cmd := send(m, key("q"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("once the session is done, q quits without asking")
	}
}

func TestPomodoroIgnoresGrowthKeysAndEarlyEnter(t *testing.T) {
	m := newPomo(newClock(), 50*time.Minute, 0, &recorder{})
	m, _ = send(m, key("+"), key("+"), key("enter"))
	if m.growth != 0 || m.pomo.phase != phaseFocus || clock2(m) != "50:00" {
		t.Errorf("+ and enter must not change a running session: %+v", m.pomo)
	}
}

func TestPomodoroSaveError(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, time.Minute, 0, &recorder{err: errors.New("disk full")})
	m, _ = send(m, size(80, 20))
	clock.add(time.Minute)
	m, cmd := clock.tick(m)
	m = runCmd(m, cmd)
	if !strings.Contains(m.View().Content, "(not saved)") {
		t.Errorf("a failed save should be visible:\n%s", lastLines(m.View().Content, 2))
	}
}

func TestPomodoroWithoutCallbacks(t *testing.T) {
	clock := newClock()
	plain := PlainTheme()
	m := New(Options{Theme: &plain, Now: clock.now, Focus: time.Minute, Break: time.Minute})
	for range 2 {
		clock.add(time.Minute)
		var cmd tea.Cmd
		m, cmd = clock.tick(m)
		m = runCmd(m, cmd)
	}
	if m.pomo.phase != phaseDone {
		t.Errorf("phase = %d, want done", m.pomo.phase)
	}
}

func TestPomodoroFooters(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, 50*time.Minute, 5*time.Minute, &recorder{})
	m, _ = send(m, size(80, 20))
	footer := func() string { return lastLines(m.View().Content, 2) }

	if f := footer(); !strings.Contains(f, "focus ▱▱▱▱▱▱▱▱▱▱ 50:00") || !strings.Contains(f, "space pause") {
		t.Errorf("focus footer:\n%s", f)
	}
	if !strings.Contains(m.View().Content, "· focus") {
		t.Error("title should say focus")
	}

	m, _ = send(m, key("q"))
	if f := footer(); !strings.Contains(f, "press q again to abandon") {
		t.Errorf("confirm footer:\n%s", f)
	}
	m, _ = send(m, key("x"))

	clock.add(50 * time.Minute)
	m, _ = clock.tick(m)
	if f := footer(); !strings.Contains(f, "break 05:00") || !strings.Contains(m.View().Content, "· break") {
		t.Errorf("break footer:\n%s", f)
	}

	clock.add(5 * time.Minute)
	m, _ = clock.tick(m)
	if f := footer(); !strings.Contains(f, "✿ session complete  1 session today") || !strings.Contains(f, "enter plant another") {
		t.Errorf("done footer:\n%s", f)
	}
}

func TestPomodoroViewGolden(t *testing.T) {
	clock := newClock()
	m := newPomo(clock, 50*time.Minute, 0, &recorder{})
	m, _ = send(m, size(70, 20))
	clock.add(32*time.Minute + 15*time.Second)
	for range 100 {
		m, _ = clock.tick(m)
	}
	assertGolden(t, "view_pomodoro", m.View().Content)
}

func TestClockFormat(t *testing.T) {
	for d, want := range map[time.Duration]string{
		50 * time.Minute:     "50:00",
		time.Second / 2:      "00:01",
		0:                    "00:00",
		-time.Second:         "00:00",
		90*time.Minute + 5e9: "90:05",
	} {
		if got := clock(d); got != want {
			t.Errorf("clock(%v) = %s, want %s", d, got, want)
		}
	}
}
