package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

func TestFormatStatus(t *testing.T) {
	s := progress.Summary{Growth: 0.72, Today: 3, FocusToday: 50, Streak: 5}
	tests := []struct {
		format, want string
	}{
		{DefaultStatusFormat, "🍂 72% · 5d"},
		{"{bar} {today}c {focus}m", "▰▰▰▰▱ 3c 50m"}, // 72% of 5 cells rounds to 4
		{"{season} {moon}", "autumn 🌓"},
		{"no placeholders", "no placeholders"},
		{"{growth}{growth}", "7272"},
	}
	for _, tt := range tests {
		got, err := formatStatus(tt.format, s, scene.Autumn, 0.25)
		if err != nil || got != tt.want {
			t.Errorf("formatStatus(%q) = %q, %v; want %q", tt.format, got, err, tt.want)
		}
	}
	if _, err := formatStatus("{nope}", s, scene.Autumn, 0); err == nil || !strings.Contains(err.Error(), "{nope}") {
		t.Errorf("unknown placeholder error = %v", err)
	}
}

func TestStatusIcon(t *testing.T) {
	for _, tt := range []struct {
		g      float64
		season scene.Season
		want   string
	}{
		{0.1, scene.Summer, "🌱"},
		{0.4, scene.Winter, "🌿"},
		{0.8, scene.Spring, "🌸"},
		{0.8, scene.Summer, "🌳"},
		{0.8, scene.Autumn, "🍂"},
		{0.8, scene.Winter, "❄️"},
	} {
		if got := statusIcon(tt.g, tt.season); got != tt.want {
			t.Errorf("statusIcon(%v, %s) = %s, want %s", tt.g, tt.season, got, tt.want)
		}
	}
}

func TestStatusCommand(t *testing.T) {
	freshDB(t)
	orig := now
	now = func() time.Time { return time.Date(2026, 11, 17, 12, 0, 0, 0, time.Local) }
	t.Cleanup(func() { now = orig })

	code, stdout, stderr := run("status")
	if code != ExitOK || stdout != "🌱 0% · 0d\n" {
		t.Errorf("empty store: exit %d, %q %q", code, stdout, stderr)
	}
	if _, stdout, _ := run("status", "--format", "{season}"); stdout != "autumn\n" {
		t.Errorf("--format {season} = %q", stdout)
	}
	if code, _, stderr := run("status", "--format", "{x}"); code != ExitUsage || !strings.Contains(stderr, "{x}") {
		t.Errorf("bad placeholder: exit %d %q", code, stderr)
	}
	if code, _, _ := run("status", "extra"); code != ExitUsage {
		t.Errorf("extra args: exit %d", code)
	}
}

// TestUsageHasNoFormatErrors guards the help text, which goes through
// Sprintf: a stray % there prints garbage like "%!·(string=…)".
func TestUsageHasNoFormatErrors(t *testing.T) {
	if strings.Contains(usage, "%!") {
		t.Errorf("usage has a formatting error:\n%s", usage)
	}
}
