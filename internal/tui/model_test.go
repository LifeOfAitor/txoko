package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

var (
	noon     = func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }
	midnight = func() time.Time { return time.Date(2026, 11, 16, 23, 30, 0, 0, time.UTC) }
)

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	golden.Assert(t, name, got)
}

func newTest(now func() time.Time) Model {
	plain := PlainTheme()
	return New(Options{Seed: 42, Growth: 0.8, Now: now, Theme: &plain})
}

// send feeds messages to the model in order and returns the final model and
// the command returned by the last message.
func send(m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func size(w, h int) tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: w, Height: h} }

func TestViewGolden(t *testing.T) {
	tests := []struct {
		name string
		now  func() time.Time
		msgs []tea.Msg
	}{
		{"view_day", noon, []tea.Msg{size(60, 18)}},
		{"view_night", midnight, []tea.Msg{size(60, 18)}},
		{"view_paused", noon, []tea.Msg{size(60, 18), key("space")}},
		{"view_narrow", noon, []tea.Msg{size(30, 12)}},
		{"view_too_small", noon, []tea.Msg{size(20, 6)}},
		{"view_garden", noon, []tea.Msg{size(60, 18), key("s")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := send(newTest(tt.now), tt.msgs...)
			golden.Assert(t, tt.name, m.View().Content)
		})
	}
}

func TestViewFillsWindowExactly(t *testing.T) {
	for _, sz := range [][2]int{{24, 9}, {60, 18}, {80, 24}, {133, 41}} {
		m, _ := send(newTest(noon), size(sz[0], sz[1]))
		lines := strings.Split(m.View().Content, "\n")
		if len(lines) != sz[1] {
			t.Errorf("%dx%d: %d lines, want %d", sz[0], sz[1], len(lines), sz[1])
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w != sz[0] {
				t.Errorf("%dx%d: line %d is %d wide, want %d: %q", sz[0], sz[1], i, w, sz[0], l)
			}
		}
	}
}

func TestViewBeforeSizeIsEmpty(t *testing.T) {
	v := newTest(noon).View()
	if v.Content != "" {
		t.Errorf("content before first resize = %q, want empty", v.Content)
	}
	if !v.AltScreen || v.WindowTitle != "txoko" {
		t.Errorf("view should use the alt screen and set the title, got %+v", v)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		_, cmd := send(newTest(noon), key(k))
		if cmd == nil {
			t.Fatalf("%s: no command returned", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s: command did not quit", k)
		}
	}
}

func TestGrowthKeys(t *testing.T) {
	m, _ := send(newTest(noon), key("+"))
	if !near(m.growth, 0.9) {
		t.Errorf("after +: growth = %v, want 0.9", m.growth)
	}
	m, _ = send(m, key("+"), key("up"), key("k"))
	if m.growth != 1 {
		t.Errorf("growth should clamp at 1, got %v", m.growth)
	}
	for range 15 {
		m, _ = send(m, key("-"))
	}
	if m.growth != 0 {
		t.Errorf("growth should clamp at 0, got %v", m.growth)
	}
}

func TestSkyCycles(t *testing.T) {
	m := newTest(noon)
	want := []bool{false, false, true, false} // auto(noon) → day → night → auto
	for i, w := range want {
		if got := m.night(); got != w {
			t.Errorf("step %d: night = %v, want %v", i, got, w)
		}
		m, _ = send(m, key("n"))
	}
	if !newTest(midnight).night() {
		t.Error("auto sky should be night at 23:30")
	}
}

func TestReseedChangesGarden(t *testing.T) {
	m, _ := send(newTest(noon), size(60, 18))
	before := m.View().Content
	m, _ = send(m, key("r"))
	if m.View().Content == before {
		t.Error("reseeding did not change the garden")
	}
	if a, b := nextSeed(1), nextSeed(1); a != b {
		t.Error("nextSeed must be deterministic")
	}
}

func TestSceneKeyCycles(t *testing.T) {
	m, _ := send(newTest(noon), size(60, 18))
	names := SceneNames()
	for i := range len(names) + 1 {
		want := names[i%len(names)]
		if !strings.Contains(m.View().Content, "· "+want+" ·") {
			t.Errorf("step %d: title should show %q", i, want)
		}
		m, _ = send(m, key("s"))
	}
}

func TestSceneOption(t *testing.T) {
	plain := PlainTheme()
	m := New(Options{Scene: "garden", Theme: &plain})
	if scenes[m.scene].Name() != "garden" {
		t.Errorf("scene = %q, want garden", scenes[m.scene].Name())
	}
	if m := New(Options{Scene: "nope"}); m.scene != 0 {
		t.Error("unknown scene should fall back to the default")
	}
	if SceneNames()[0] != "bonsai" {
		t.Error("bonsai should be the default scene")
	}
}

func TestTickAdvancesFrame(t *testing.T) {
	m := newTest(noon)
	m, cmd := send(m, tickMsg{id: m.tickID})
	if m.frame != 1 {
		t.Errorf("frame = %d, want 1", m.frame)
	}
	if cmd == nil {
		t.Error("tick should schedule the next tick")
	}
}

func TestPauseStopsAndResumesTicking(t *testing.T) {
	m, _ := send(newTest(noon), key("space"))
	if !m.paused {
		t.Fatal("space should pause")
	}
	m, cmd := send(m, tickMsg{id: m.tickID})
	if m.frame != 0 || cmd != nil {
		t.Error("a paused model must not advance or keep ticking")
	}

	oldID := m.tickID
	m, cmd = send(m, key("p"))
	if m.paused || cmd == nil {
		t.Fatal("p should resume and restart the tick loop")
	}
	// A stale tick from the old loop must be ignored...
	m, cmd = send(m, tickMsg{id: oldID})
	if m.frame != 0 || cmd != nil {
		t.Error("stale tick was not ignored")
	}
	// ...while the new loop's ticks count.
	m, _ = send(m, tickMsg{id: m.tickID})
	if m.frame != 1 {
		t.Errorf("frame = %d, want 1", m.frame)
	}
}

func TestInitStartsTicking(t *testing.T) {
	if newTest(noon).Init() == nil {
		t.Error("Init should start the tick loop")
	}
}

func TestUnknownMessagesAreIgnored(t *testing.T) {
	m := newTest(noon)
	next, cmd := send(m, struct{}{}, key("x"))
	if cmd != nil || next.growth != m.growth || next.frame != m.frame {
		t.Error("unknown input should change nothing")
	}
}

func TestNewDefaults(t *testing.T) {
	m := New(Options{Growth: 7})
	if m.seed == 0 || m.fps != DefaultFPS || m.now == nil || m.growth != 1 {
		t.Errorf("unexpected defaults: seed=%d fps=%d growth=%v", m.seed, m.fps, m.growth)
	}
	if first, second := DefaultSeed(), DefaultSeed(); first != second {
		t.Error("DefaultSeed must be stable")
	}
}

func TestClamp01(t *testing.T) {
	nan := 0.0
	nan /= nan
	for in, want := range map[float64]float64{-1: 0, 0.5: 0.5, 2: 1, nan: 0} {
		if got := clamp01(in); got != want {
			t.Errorf("clamp01(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestRenderCanvasGroupsRuns(t *testing.T) {
	c := canvas.New(5, 1)
	c.Set(0, 0, 'a', canvas.InkStem)
	c.Set(1, 0, 'b', canvas.InkStem)
	c.Set(3, 0, 'c', canvas.InkStar)

	got := renderCanvas(c, PlainTheme())[0]
	if got != "ab c " {
		t.Errorf("plain render = %q, want %q", got, "ab c ")
	}
	// With a real theme the styled output must still be 5 cells wide.
	if w := lipgloss.Width(renderCanvas(c, DefaultTheme())[0]); w != 5 {
		t.Errorf("styled width = %d, want 5", w)
	}
}

func TestFooterLineFallbacks(t *testing.T) {
	rich := layout{[]span{{text: "left"}}, []span{{text: "right"}}}
	compact := layout{[]span{{text: "l"}}, nil}
	tests := []struct {
		inner int
		want  string
	}{
		{20, " left         right "}, // rich layout fits
		{8, " l      "},              // falls back to the compact layout
		{2, "  "},                    // nothing fits: blank line
	}
	for _, tt := range tests {
		got := footerLine(tt.inner, rich, compact)
		if got != tt.want {
			t.Errorf("inner %d: %q, want %q", tt.inner, got, tt.want)
		}
		if w := lipgloss.Width(got); w != tt.inner {
			t.Errorf("inner %d: width %d", tt.inner, w)
		}
	}
}

func TestFrameWithoutTitleRoom(t *testing.T) {
	got := frame(4, []span{{text: "too long title"}}, []string{"abcd"}, "    ", PlainTheme())
	want := "╭────╮\n│abcd│\n├────┤\n│    │\n╰────╯"
	if got != want {
		t.Errorf("frame =\n%s\nwant\n%s", got, want)
	}
}

func TestProgressBar(t *testing.T) {
	for _, tt := range []struct {
		v           float64
		full, empty int
	}{{0, 0, 4}, {0.5, 2, 2}, {1, 4, 0}, {2, 4, 0}, {-1, 0, 4}} {
		f, e := progressBar(tt.v, 4)
		if len([]rune(f)) != tt.full || len([]rune(e)) != tt.empty {
			t.Errorf("progressBar(%v) = %q %q", tt.v, f, e)
		}
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }
