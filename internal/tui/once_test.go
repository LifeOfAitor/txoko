package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/progress"
)

func once(t *testing.T, o OnceOptions) string {
	t.Helper()
	if o.Now == nil {
		o.Now = noon
	}
	if o.Seed == 0 {
		o.Seed = 42
	}
	if o.MoonPhase == nil {
		o.MoonPhase = &testMoon
	}
	var b bytes.Buffer
	if err := Once(&b, o); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestOnceGolden(t *testing.T) {
	assertGolden(t, "once_plain", once(t, OnceOptions{Growth: 1}))
	assertGolden(t, "once_activity", once(t, OnceOptions{
		Growth: 0.72, Now: midnight, Season: "winter",
		Summary: &progress.Summary{Growth: 0.72, Today: 3, FocusToday: 50, Streak: 5},
	}))
}

func TestOnceInfo(t *testing.T) {
	out := once(t, OnceOptions{Growth: 1, Summary: &progress.Summary{Today: 1, Streak: 2}})
	for _, want := range []string{"txoko · bonsai · autumn · day", "growth ▰▰▰▰▰▰▰▰▰▰ 100%", "today  1 commit", "streak 2 days", "date   Mon 16 Nov"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(once(t, OnceOptions{Growth: 1}), "streak") {
		t.Error("without a summary there should be no activity lines")
	}
	focus := once(t, OnceOptions{Growth: 1, Summary: &progress.Summary{FocusToday: 25}})
	if !strings.Contains(focus, "0 commits · 25 min focus") {
		t.Errorf("focus minutes missing:\n%s", focus)
	}
}

func TestOnceHasNoMoonPhaseLine(t *testing.T) {
	if out := once(t, OnceOptions{Growth: 1}); strings.Contains(out, "moon") || strings.Contains(out, "quarter") {
		t.Errorf("the info panel should not mention the moon phase:\n%s", out)
	}
}

func TestOncePlainHasNoEscapes(t *testing.T) {
	for _, now := range []func() time.Time{noon, midnight} { // nights have dimmed cells
		if out := once(t, OnceOptions{Growth: 1, Now: now}); strings.Contains(out, "\x1b[") {
			t.Error("plain output must not contain ANSI escapes")
		}
	}
	if out := once(t, OnceOptions{Growth: 1, Color: true}); !strings.Contains(out, "\x1b[") {
		t.Error("colored output should contain ANSI escapes")
	}
}

func TestOnceSizeAndDefaults(t *testing.T) {
	small := once(t, OnceOptions{Growth: 1, Width: 20, Height: 8})
	if n := strings.Count(small, "\n"); n != 8 {
		t.Errorf("a 20x8 scene should print 8 lines, got %d", n)
	}
	if n := strings.Count(once(t, OnceOptions{Growth: 1}), "\n"); n != OnceHeight {
		t.Errorf("default size should print %d lines, got %d", OnceHeight, n)
	}
	var b bytes.Buffer
	if err := Once(&b, OnceOptions{Growth: 1}); err != nil || b.Len() == 0 {
		t.Errorf("zero options should still draw: %v", err)
	}
	if !strings.Contains(once(t, OnceOptions{Growth: 1, Scene: "garden"}), "· garden ·") {
		t.Error("scene option ignored")
	}
}

func TestOnceNoTrailingSpaces(t *testing.T) {
	for _, line := range strings.Split(once(t, OnceOptions{Growth: 1}), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("line has trailing spaces: %q", line)
		}
	}
}

func TestOnceSky(t *testing.T) {
	if !strings.Contains(once(t, OnceOptions{Growth: 1, Sky: "night"}), "· night") {
		t.Error("--sky night should make the splash a night scene at noon")
	}
	if !strings.Contains(once(t, OnceOptions{Growth: 1, Sky: "day", Now: midnight}), "· day") {
		t.Error("--sky day should make the splash a day scene at midnight")
	}
}
