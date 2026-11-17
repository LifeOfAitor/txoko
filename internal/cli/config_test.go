package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withConfig points txoko at a settings file with the given body.
func withConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if body != "" {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TXOKO_CONFIG", p)
	return p
}

func TestConfigProvidesDefaults(t *testing.T) {
	freshDB(t)
	withConfig(t, `
scene = "garden"
season = "winter"
seed = 9
work = true
mini = true
[pomodoro]
break = 10
`)
	calls := fakeTUI(t, nil)
	if code, _, stderr := run(); code != ExitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	o := (*calls)[0]
	if o.Scene != "garden" || o.Season != "winter" || o.Seed != 9 || !o.Mini || o.Source == nil {
		t.Errorf("config not applied: %+v", o)
	}

	calls = fakeTUI(t, nil)
	run("--pomodoro", "25")
	if o := (*calls)[0]; o.Break != 10*time.Minute || o.Source != nil {
		t.Errorf("pomodoro should use the configured break and leave work mode: break %v, work %v", o.Break, o.Source != nil)
	}
}

func TestFlagsBeatConfig(t *testing.T) {
	freshDB(t)
	withConfig(t, "scene = \"garden\"\nwork = true\nmini = true\n")
	calls := fakeTUI(t, nil)
	run("--scene", "bonsai", "--work=false", "--mini=false")
	if o := (*calls)[0]; o.Scene != "bonsai" || o.Source != nil || o.Mini {
		t.Errorf("flags should win: %+v", o)
	}

	calls = fakeTUI(t, nil)
	if code, _, stderr := run("--growth", "0.5"); code != ExitOK || (*calls)[0].Source != nil {
		t.Errorf("--growth should beat work = true from the config: exit %d %s", code, stderr)
	}
	if code, _, _ := run("--work", "--growth", "0.5"); code != ExitUsage {
		t.Error("--work and --growth together on the command line still clash")
	}
}

func TestConfigForOnceAndStatus(t *testing.T) {
	freshDB(t)
	withConfig(t, "[once]\nsize = \"20x8\"\nwork = true\n[status]\nformat = \"{season}!\"\n")
	_, stdout, _ := run("--once")
	if n := strings.Count(stdout, "\n"); n != 8 || !strings.Contains(stdout, "streak") {
		t.Errorf("once should use the configured size and activity:\n%s", stdout)
	}
	if _, stdout, _ := run("status"); !strings.HasSuffix(stdout, "!\n") {
		t.Errorf("status should use the configured format: %q", stdout)
	}
	if _, stdout, _ := run("status", "--format", "{growth}"); stdout != "0\n" {
		t.Errorf("--format beats the config: %q", stdout)
	}
}

func TestInvalidConfig(t *testing.T) {
	tests := []struct{ body, want string }{
		{`scene = "volcano"`, `unknown scene "volcano"`},
		{`season = "monsoon"`, `unknown season "monsoon"`},
		{"[once]\nsize = \"huge\"", "once.size"},
		{"[pomodoro]\nbreak = 500", "pomodoro.break"},
		{"[status]\nformat = \"{nope}\"", "status.format"},
		{`sene = "bonsai"`, "unknown setting sene"},
		{`work = "yes"`, "config"},
	}
	for _, tt := range tests {
		path := withConfig(t, tt.body)
		for _, args := range [][]string{nil, {"status"}} {
			fakeTUI(t, nil)
			code, _, stderr := run(args...)
			if code != ExitError || !strings.Contains(stderr, tt.want) || !strings.Contains(stderr, path) {
				t.Errorf("%q %v: exit %d, stderr %q; want %q and the path", tt.body, args, code, stderr, tt.want)
			}
		}
	}
}

func TestConfigCommand(t *testing.T) {
	path := withConfig(t, "")
	_, stdout, _ := run("config")
	if !strings.Contains(stdout, path) || !strings.Contains(stdout, "not created yet") {
		t.Errorf("config before init:\n%s", stdout)
	}
	if code, stdout, _ := run("config", "init"); code != ExitOK || !strings.Contains(stdout, "created") {
		t.Fatalf("config init: exit %d %s", code, stdout)
	}
	if _, stdout, _ := run("config"); !strings.Contains(stdout, "✓ valid") {
		t.Errorf("config after init:\n%s", stdout)
	}
	if _, stdout, _ := run("config", "init"); !strings.Contains(stdout, "already exists") {
		t.Errorf("a second init must not overwrite:\n%s", stdout)
	}
	if code, _, _ := run("config", "edit"); code != ExitUsage {
		t.Error("unknown config subcommand should be a usage error")
	}

	// The written template must leave txoko's defaults unchanged.
	calls := fakeTUI(t, nil)
	run()
	if o := (*calls)[0]; o.Scene != "bonsai" || o.Source != nil || o.Mini || o.Seed != 0 {
		t.Errorf("the template changed the defaults: %+v", o)
	}

	withConfig(t, `scene = "volcano"`)
	if code, _, _ := run("config"); code != ExitError {
		t.Error("`txoko config` should report an invalid file")
	}
	t.Setenv("TXOKO_CONFIG", "/dev/null/config.toml")
	if code, _, _ := run("config", "init"); code != ExitError {
		t.Error("init into an impossible folder should fail")
	}
}
