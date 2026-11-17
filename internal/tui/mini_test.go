package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func newMini(now func() time.Time) Model {
	plain := PlainTheme()
	return New(Options{Seed: 42, Growth: 0.8, Now: now, Theme: &plain, Mini: true})
}

func TestMiniFillsTheCornerExactly(t *testing.T) {
	for _, sz := range [][2]int{{12, 4}, {30, 10}, {60, 20}} {
		m, _ := send(newMini(noon), size(sz[0], sz[1]))
		for _, keys := range []bool{false, true} {
			if keys {
				m, _ = send(m, key("?"))
			}
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != sz[1] {
				t.Errorf("%dx%d keys=%v: %d lines, want %d", sz[0], sz[1], keys, len(lines), sz[1])
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w != sz[0] {
					t.Errorf("%dx%d keys=%v: line %d is %d wide", sz[0], sz[1], keys, i, w)
				}
			}
		}
	}
}

func TestMiniHasNoFrameAndAKeysToggle(t *testing.T) {
	m, _ := send(newMini(noon), size(40, 12))
	view := m.View().Content
	if strings.ContainsAny(view, "╭╮╰╯│") {
		t.Errorf("mini mode should have no frame:\n%s", view)
	}
	if !strings.Contains(lastLines(view, 1), "? keys") || strings.Contains(view, "q quit") {
		t.Errorf("collapsed: one status line with '? keys':\n%s", lastLines(view, 2))
	}

	m, _ = send(m, key("?"))
	expanded := m.View().Content
	for _, k := range []string{"s scene", "n night", "q quit", "? hide keys"} {
		if !strings.Contains(expanded, k) {
			t.Errorf("? should list %q:\n%s", k, lastLines(expanded, 4))
		}
	}
	m, _ = send(m, key("?"))
	if strings.Contains(m.View().Content, "q quit") {
		t.Error("a second ? should hide the keys again")
	}
}

func TestMiniMinimumSize(t *testing.T) {
	m, _ := send(newMini(noon), size(12, 4))
	if strings.Contains(m.View().Content, "more room") {
		t.Error("12x4 is enough for mini mode")
	}
	m, _ = send(newMini(noon), size(10, 3))
	if !strings.Contains(m.View().Content, "more room") {
		t.Error("10x3 is too small even for mini mode")
	}
}

func TestMiniPomodoro(t *testing.T) {
	plain := PlainTheme()
	clock := newClock()
	m := New(Options{Seed: 42, Theme: &plain, Now: clock.now, Mini: true, Focus: 25 * time.Minute})
	m, _ = send(m, size(40, 10))
	if !strings.Contains(lastLines(m.View().Content, 1), "focus") || !strings.Contains(m.View().Content, "25:00") {
		t.Errorf("mini pomodoro should show the timer:\n%s", lastLines(m.View().Content, 1))
	}
}

func TestMiniGolden(t *testing.T) {
	m, _ := send(newMini(midnight), size(36, 12))
	assertGolden(t, "view_mini", m.View().Content)
	m, _ = send(m, key("?"))
	assertGolden(t, "view_mini_keys", m.View().Content)
}
