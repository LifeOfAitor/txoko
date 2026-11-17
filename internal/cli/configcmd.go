package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/LifeOfAitor/txoko/internal/config"
	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/tui"
)

// loadConfig reads the settings file and checks every value, so a mistake
// is reported once, clearly, instead of surfacing as an odd flag error.
func loadConfig() (config.Config, error) {
	path, err := config.Path()
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("config %w", err)
	}
	bad := func(format string, a ...any) error {
		return fmt.Errorf("config %s: %s", path, fmt.Sprintf(format, a...))
	}
	if cfg.Scene != "" && !slices.Contains(tui.SceneNames(), cfg.Scene) {
		return cfg, bad("unknown scene %q (available: %s)", cfg.Scene, strings.Join(tui.SceneNames(), ", "))
	}
	if _, ok := scene.ParseSeason(cfg.Season); cfg.Season != "" && !ok {
		return cfg, bad("unknown season %q (use spring, summer, autumn or winter, or leave it empty)", cfg.Season)
	}
	if cfg.Once.Size != "" {
		var size sceneSize
		if err := size.Set(cfg.Once.Size); err != nil {
			return cfg, bad("once.size %q: %v", cfg.Once.Size, err)
		}
	}
	if b := (minutes{lo: 0, hi: 120}); b.Set(strconv.Itoa(cfg.Pomodoro.Break)) != nil {
		return cfg, bad("pomodoro.break must be between 0 and 120 minutes, got %d", cfg.Pomodoro.Break)
	}
	if cfg.Status.Format != "" {
		if _, err := formatStatus(cfg.Status.Format, progress.Summary{}, scene.Summer); err != nil {
			return cfg, bad("status.format: %v", err)
		}
	}
	return cfg, nil
}

// runConfig shows where the settings file lives, or creates it.
func runConfig(args []string, stdout, stderr io.Writer) int {
	path, err := config.Path()
	if err != nil {
		return fail(stderr, err)
	}
	switch {
	case len(args) == 0:
		fmt.Fprintln(stdout, path)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintln(stdout, "  not created yet — `txoko config init` writes one with every setting explained")
			return ExitOK
		}
		if _, err := loadConfig(); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, "  ✓ valid")
		return ExitOK
	case len(args) == 1 && args[0] == "init":
		err := config.Init(path)
		if errors.Is(err, config.ErrExists) {
			fmt.Fprintf(stdout, "%s already exists — edit it with: $EDITOR %s\n", path, path)
			return ExitOK
		}
		if err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "✓ created %s\n  every setting is explained inside; flags always win over it\n", path)
		return ExitOK
	}
	return usageError(stderr, "usage: txoko config [init]")
}
