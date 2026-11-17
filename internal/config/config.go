// Package config reads txoko's optional settings file,
// ~/.config/txoko/config.toml. Every setting has a command-line flag that
// overrides it; the file just saves typing the same flags every time.
package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config mirrors the file. The zero value means "nothing configured".
type Config struct {
	Scene  string `toml:"scene"`
	Season string `toml:"season"`
	Seed   uint64 `toml:"seed"`
	Work   bool   `toml:"work"`

	Once struct {
		Size string `toml:"size"`
		Work bool   `toml:"work"`
	} `toml:"once"`

	Pomodoro struct {
		Break int `toml:"break"`
	} `toml:"pomodoro"`

	Status struct {
		Format string `toml:"format"`
	} `toml:"status"`
}

// Template is the commented file `txoko config init` writes. It spells out
// the built-in defaults, so writing it changes nothing until you edit it.
//
//go:embed template.toml
var Template string

// Path returns where the file lives: $TXOKO_CONFIG if set, otherwise
// $XDG_CONFIG_HOME/txoko/config.toml, falling back to ~/.config.
func Path() (string, error) {
	if p := os.Getenv("TXOKO_CONFIG"); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "txoko", "config.toml"), nil
}

// Load reads the file at path. A missing file is not an error: it simply
// means no settings. Unknown keys are reported, so a typo like "sene" doesn't
// get silently ignored.
func Load(path string) (Config, error) {
	var c Config
	md, err := toml.DecodeFile(path, &c)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if extra := md.Undecoded(); len(extra) > 0 {
		keys := make([]string, len(extra))
		for i, k := range extra {
			keys[i] = k.String()
		}
		return Config{}, fmt.Errorf("%s: unknown setting %s", path, strings.Join(keys, ", "))
	}
	return c, nil
}

// ErrExists is returned by Init when the file is already there.
var ErrExists = errors.New("config file already exists")

// Init writes Template to path, creating its folder, and never overwrites
// an existing file.
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%s: %w", path, ErrExists)
	}
	if err != nil {
		return err
	}
	if _, err := f.WriteString(Template); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
