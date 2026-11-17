package cli

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// DefaultStatusFormat is what `txoko status` prints without --format.
const DefaultStatusFormat = "{icon} {growth}% · {streak}d"

// statusFields documents the placeholders --format understands.
var statusFields = []string{"icon", "growth", "bar", "today", "focus", "streak", "season", "moon"}

var placeholder = regexp.MustCompile(`\{([a-z]+)\}`)

// runStatus prints a single line for a tmux status bar or a shell prompt.
// It is meant to be called every few seconds, so it only reads.
func runStatus(args []string, stdout, stderr io.Writer) int {
	cfg, err := loadConfig()
	if err != nil {
		return fail(stderr, err)
	}
	fs := newFlagSet("status")
	format := fs.String("format", cmp.Or(cfg.Status.Format, DefaultStatusFormat), "")
	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if fs.NArg() > 0 {
		return usageError(stderr, "usage: txoko status [--format FORMAT]")
	}

	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()
	s, err := summarize(context.Background(), st)
	if err != nil {
		return fail(stderr, err)
	}

	t := now()
	line, err := formatStatus(*format, s, scene.SeasonOf(t), scene.MoonPhaseOf(t))
	if err != nil {
		return usageError(stderr, err.Error())
	}
	fmt.Fprintln(stdout, line)
	return ExitOK
}

// formatStatus fills the placeholders of format.
func formatStatus(format string, s progress.Summary, season scene.Season, moonPhase float64) (string, error) {
	_, moon := scene.MoonPhaseName(moonPhase)
	filled := int(s.Growth*5 + 0.5)
	values := map[string]string{
		"icon":   statusIcon(s.Growth, season),
		"growth": fmt.Sprintf("%.0f", s.Growth*100),
		"bar":    strings.Repeat("▰", filled) + strings.Repeat("▱", 5-filled),
		"today":  fmt.Sprint(s.Today),
		"focus":  fmt.Sprint(s.FocusToday),
		"streak": fmt.Sprint(s.Streak),
		"season": season.String(),
		"moon":   moon,
	}

	var unknown string
	line := placeholder.ReplaceAllStringFunc(format, func(m string) string {
		key := m[1 : len(m)-1]
		v, ok := values[key]
		if !ok && unknown == "" {
			unknown = key
		}
		return v
	})
	if unknown != "" {
		return "", fmt.Errorf("unknown placeholder {%s} (use: {%s})", unknown, strings.Join(statusFields, "}, {"))
	}
	return line, nil
}

// statusIcon grows with the tree: a sprout, a young plant, then the season's
// look once it's well grown.
func statusIcon(growth float64, season scene.Season) string {
	switch {
	case growth < 0.2:
		return "🌱"
	case growth < 0.6:
		return "🌿"
	}
	switch season {
	case scene.Spring:
		return "🌸"
	case scene.Autumn:
		return "🍂"
	case scene.Winter:
		return "❄️"
	}
	return "🌳"
}
