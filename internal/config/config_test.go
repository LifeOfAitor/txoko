package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissingFileMeansNoSettings(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || c != (Config{}) {
		t.Errorf("Load(missing) = %+v, %v", c, err)
	}
}

func TestLoad(t *testing.T) {
	c, err := Load(write(t, `
scene = "garden"
season = "winter"
seed = 7
work = true
mini = true
[once]
size = "40x16"
work = true
[pomodoro]
break = 10
[status]
format = "{icon}"
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Scene != "garden" || c.Season != "winter" || c.Seed != 7 || !c.Work || !c.Mini ||
		c.Once.Size != "40x16" || !c.Once.Work || c.Pomodoro.Break != 10 || c.Status.Format != "{icon}" {
		t.Errorf("Load = %+v", c)
	}
}

func TestUnknownSettingIsReported(t *testing.T) {
	_, err := Load(write(t, "sene = \"bonsai\"\n[status]\nformt = \"x\"\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown setting sene, status.formt") {
		t.Errorf("err = %v", err)
	}
}

func TestBadTomlIsReported(t *testing.T) {
	p := write(t, "work = \"yes please\"\n")
	if _, err := Load(p); err == nil || !strings.Contains(err.Error(), p) {
		t.Errorf("a wrong type should name the file: %v", err)
	}
}

// TestTemplateMatchesDefaults: writing the template must not change how
// txoko behaves, so its values are the built-in defaults.
func TestTemplateMatchesDefaults(t *testing.T) {
	c, err := Load(write(t, Template))
	if err != nil {
		t.Fatal(err)
	}
	if c.Scene != "bonsai" || c.Season != "" || c.Seed != 0 || c.Work || c.Mini ||
		c.Once.Size != "34x14" || c.Once.Work || c.Pomodoro.Break != 0 ||
		c.Status.Format != "{icon} {growth}% · {streak}d" {
		t.Errorf("template = %+v", c)
	}
}

func TestInit(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deep", "txoko", "config.toml")
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if string(body) != Template {
		t.Error("Init should write the template")
	}
	if err := Init(p); !errors.Is(err, ErrExists) {
		t.Errorf("second Init = %v, want ErrExists", err)
	}
	if err := Init("/dev/null/x/config.toml"); err == nil {
		t.Error("Init into an impossible folder should fail")
	}
}

func TestPath(t *testing.T) {
	t.Setenv("TXOKO_CONFIG", "/tmp/my.toml")
	if p, _ := Path(); p != "/tmp/my.toml" {
		t.Errorf("TXOKO_CONFIG ignored: %s", p)
	}
	t.Setenv("TXOKO_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	if p, _ := Path(); p != "/cfg/txoko/config.toml" {
		t.Errorf("XDG path = %s", p)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/me")
	if p, _ := Path(); p != "/home/me/.config/txoko/config.toml" {
		t.Errorf("fallback path = %s", p)
	}
}
