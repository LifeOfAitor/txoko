package cli

import (
	"strings"
	"testing"
	"time"
)

func TestPomodoroFlags(t *testing.T) {
	calls := fakeTUI(t, nil)
	if code, _, stderr := run("--pomodoro", "50", "--break", "10"); code != ExitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	o := (*calls)[0]
	if o.Focus != 50*time.Minute || o.Break != 10*time.Minute || o.OnFocusDone == nil || o.OnBreakDone == nil {
		t.Errorf("options = focus %v break %v", o.Focus, o.Break)
	}
	if o.Source != nil {
		t.Error("a pomodoro grows its own tree, not the activity one")
	}
}

func TestPomodoroFlagErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		// Not whole minutes: decimals, words, empty.
		{[]string{"--pomodoro", "0.5"}, `invalid value "0.5" for flag -pomodoro: use whole minutes, like 25`},
		{[]string{"--pomodoro", "25.5"}, "use whole minutes"},
		{[]string{"--pomodoro", "abc"}, "use whole minutes"},
		{[]string{"--pomodoro", ""}, "use whole minutes"},
		{[]string{"--pomodoro=1e3"}, "use whole minutes"},
		// Out of range: zero, negative, too long.
		{[]string{"--pomodoro", "0"}, "use between 1 and 240 minutes"},
		{[]string{"--pomodoro", "-3"}, "use between 1 and 240 minutes"},
		{[]string{"--pomodoro=-3"}, "use between 1 and 240 minutes"},
		{[]string{"--pomodoro", "241"}, "use between 1 and 240 minutes"},
		// The break gets the same treatment.
		{[]string{"--pomodoro", "25", "--break", "0.5"}, `invalid value "0.5" for flag -break: use whole minutes`},
		{[]string{"--pomodoro", "25", "--break", "-3"}, "use between 0 and 120 minutes"},
		{[]string{"--pomodoro", "25", "--break", "121"}, "use between 0 and 120 minutes"},
		{[]string{"--pomodoro", "25", "--break", "five"}, "use whole minutes"},
		{[]string{"--break", "5"}, "only works together with --pomodoro"},
		{[]string{"--pomodoro", "25", "--work"}, "can't be combined with --work"},
		{[]string{"--pomodoro", "25", "--growth", "0.5"}, "can't be combined with --growth"},
	}
	for _, tt := range tests {
		fakeTUI(t, nil)
		code, _, stderr := run(tt.args...)
		if code != ExitUsage || !strings.Contains(stderr, tt.want) {
			t.Errorf("%v: exit %d, stderr %q; want %q", tt.args, code, stderr, tt.want)
		}
	}
}

// TestPomodoroLimits checks the exact edges that are allowed.
func TestPomodoroLimits(t *testing.T) {
	for _, tt := range []struct {
		args         []string
		focus, pause time.Duration
	}{
		{[]string{"--pomodoro", "1"}, time.Minute, 0},
		{[]string{"--pomodoro", "240", "--break", "120"}, 240 * time.Minute, 120 * time.Minute},
		{[]string{"--pomodoro", " 25 ", "--break", "0"}, 25 * time.Minute, 0},
	} {
		calls := fakeTUI(t, nil)
		if code, _, stderr := run(tt.args...); code != ExitOK {
			t.Errorf("%v: exit %d %s", tt.args, code, stderr)
			continue
		}
		if o := (*calls)[0]; o.Focus != tt.focus || o.Break != tt.pause {
			t.Errorf("%v: focus %v break %v", tt.args, o.Focus, o.Break)
		}
	}
}

func TestMinutesString(t *testing.T) {
	m := minutes{n: 25}
	if m.String() != "25" {
		t.Errorf("String() = %q", m.String())
	}
}

func TestFinishedSessionIsRecordedAndNotified(t *testing.T) {
	freshDB(t)
	var notes []string
	orig := notifySend
	notifySend = func(_, msg string) error { notes = append(notes, msg); return nil }
	t.Cleanup(func() { notifySend = orig })

	calls := fakeTUI(t, nil)
	run("--pomodoro", "50", "--break", "5")
	o := (*calls)[0]

	if err := o.OnFocusDone(50 * time.Minute); err != nil {
		t.Fatal(err)
	}
	o.OnBreakDone()
	if len(notes) != 2 || !strings.Contains(notes[0], "50 minutes of focus") || !strings.Contains(notes[1], "Break's over") {
		t.Errorf("notifications = %q", notes)
	}
	if _, stdout, _ := run("stats"); !strings.Contains(stdout, "50 min of focus") {
		t.Errorf("the session should count in stats:\n%s", stdout)
	}
}

func TestRecordFocusFailsWithoutStore(t *testing.T) {
	t.Setenv("TXOKO_DB", "/dev/null/txoko.db")
	if err := recordFocus(25 * time.Minute); err == nil {
		t.Error("recordFocus should report a store that can't open")
	}
}
