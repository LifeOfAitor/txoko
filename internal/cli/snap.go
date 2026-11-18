package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/LifeOfAitor/txoko/internal/tui"
)

// runSnap saves the scene as an SVG picture: for a README, a wallpaper or
// to show someone. Like --once it's fully grown unless --growth or --work
// say otherwise.
func runSnap(args []string, stdout, stderr io.Writer) int {
	cfg, err := loadConfig()
	if err != nil {
		return fail(stderr, err)
	}
	fs := newFlagSet("snap")
	opts := tui.Options{Growth: 1}
	var work bool
	output := "txoko.svg"
	fs.StringVar(&output, "o", output, "")
	fs.StringVar(&output, "output", output, "")
	fs.BoolVar(&work, "work", false, "")
	fs.Float64Var(&opts.Growth, "growth", 1, "")
	fs.Uint64Var(&opts.Seed, "seed", cfg.Seed, "")
	fs.StringVar(&opts.Scene, "scene", cfg.Scene, "")
	fs.StringVar(&opts.Season, "season", cfg.Season, "")
	fs.StringVar(&opts.Sky, "sky", "", "")
	size := sceneSize{w: tui.SnapWidth, h: tui.SnapHeight}
	fs.Var(&size, "size", "")
	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if fs.NArg() > 0 {
		return usageError(stderr, "usage: txoko snap [-o FILE] [--scene S] [--season S] [--sky S] [--growth N | --work] [--seed N] [--size WxH]")
	}
	if msg := checkLook(opts); msg != "" {
		return usageError(stderr, msg)
	}
	if work && flagSet(fs, "growth") {
		return usageError(stderr, "--growth sets the scene by hand, so it can't be combined with --work")
	}

	o := tui.OnceOptions{
		Scene: opts.Scene, Season: opts.Season, Sky: opts.Sky, Seed: opts.Seed,
		Growth: opts.Growth, Width: size.w, Height: size.h, Now: now,
	}
	if work {
		st, err := openStore()
		if err != nil {
			return fail(stderr, err)
		}
		defer func() { _ = st.Close() }()
		s, err := summarize(context.Background(), st)
		if err != nil {
			return fail(stderr, err)
		}
		o.Growth = s.Growth
	}

	var svg bytes.Buffer
	_ = tui.Snap(&svg, o, tui.Frappe) // writing to memory can't fail
	if output == "-" {
		_, err = stdout.Write(svg.Bytes())
	} else {
		err = os.WriteFile(output, svg.Bytes(), 0o644)
	}
	if err != nil {
		return fail(stderr, fmt.Errorf("saving the picture: %w", err))
	}
	if output != "-" {
		fmt.Fprintf(stdout, "saved %s (%d×%d)\n", output, size.w, size.h)
	}
	return ExitOK
}
