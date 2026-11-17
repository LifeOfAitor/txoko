package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/store"
)

// fakeGitHub stands in for GitHub's API while a test runs, with a token
// to go with it. It counts how often it's asked.
func fakeGitHub(t *testing.T, answer string) *atomic.Int32 {
	t.Helper()
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		if r.Header.Get("Authorization") != "bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(answer))
	}))
	origAPI, origToken := githubAPI, githubToken
	githubAPI = srv.URL
	githubToken = func() (string, error) { return "test-token", nil }
	t.Cleanup(func() {
		srv.Close()
		githubAPI, githubToken = origAPI, origToken
	})
	return &asked
}

const yearJSON = `{"data": {"viewer": {"login": "octo", "contributionsCollection": {"contributionCalendar": {"weeks": [
	{"contributionDays": [{"date": "2026-11-01", "contributionCount": 2}, {"date": "2026-11-02", "contributionCount": 5}]},
	{"contributionDays": [{"date": "2026-11-08", "contributionCount": 1}, {"date": "2026-11-09", "contributionCount": 0}]}
]}}}}}`

// yearTest gives the test its own empty database and a fixed clock.
func yearTest(t *testing.T) {
	t.Helper()
	freshDB(t)
	orig := now
	now = func() time.Time { return time.Date(2026, 11, 17, 12, 0, 0, 0, time.Local) }
	t.Cleanup(func() { now = orig })
}

func TestYearFromGitHub(t *testing.T) {
	yearTest(t)
	asked := fakeGitHub(t, yearJSON)
	code, out, errOut := run("year")
	if code != ExitOK || errOut != "" {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"@octo on GitHub", "8 contributions", "7 in the week of 1 Nov", "2 days at most", "just now"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	// Asked again soon after, it answers from its cache.
	if run("year"); asked.Load() != 1 {
		t.Errorf("GitHub was asked %d times, want 1", asked.Load())
	}
	// Unless told to refresh.
	if run("year", "--refresh"); asked.Load() != 2 {
		t.Errorf("--refresh should ask GitHub again (asked %d times)", asked.Load())
	}
}

func TestYearOfAnotherUser(t *testing.T) {
	yearTest(t)
	fakeGitHub(t, strings.Replace(strings.Replace(yearJSON, "viewer", "user", 1), "octo", "hubot", 1))
	if code, out, _ := run("year", "--user", "hubot"); code != ExitOK || !strings.Contains(out, "@hubot on GitHub") {
		t.Errorf("code %d:\n%s", code, out)
	}
}

func TestYearOffline(t *testing.T) {
	yearTest(t)
	// No token and nothing saved: explain, and point to --local.
	code, _, errOut := run("year")
	if code != ExitError || !strings.Contains(errOut, "GITHUB_TOKEN") || !strings.Contains(errOut, "--local") {
		t.Errorf("code %d, stderr %q", code, errOut)
	}

	// With an older calendar saved, an unreachable GitHub isn't fatal.
	fakeGitHub(t, yearJSON)
	run("year")
	githubAPI = "http://127.0.0.1:1/graphql" // GitHub goes away
	orig := now
	now = func() time.Time { return orig().Add(48 * time.Hour) }
	code, out, errOut := run("year")
	if code != ExitOK || !strings.Contains(errOut, "couldn't update from GitHub") || !strings.Contains(out, "2 days ago") {
		t.Errorf("code %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestYearLocal(t *testing.T) {
	yearTest(t)
	st, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i, at := range []time.Time{
		now().Add(-time.Hour), now().Add(-25 * time.Hour), now().Add(-26 * time.Hour), now().AddDate(-2, 0, 0),
	} {
		if _, err := st.Add(ctx, store.Event{Kind: store.KindCommit, Repo: "r", Ref: string(rune('a' + i)), At: at}); err != nil {
			t.Fatal(err)
		}
	}
	_ = st.Close()

	code, out, _ := run("year", "--local")
	if code != ExitOK {
		t.Fatalf("code %d", code)
	}
	for _, want := range []string{"txoko's own record", "3 contributions", "2 days at most"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestYearUsage(t *testing.T) {
	for _, args := range [][]string{
		{"year", "extra"},
		{"year", "--local", "--user", "x"},
		{"year", "--local", "--refresh"},
		{"year", "--nope"},
	} {
		if code, _, _ := run(args...); code != ExitUsage {
			t.Errorf("%v: code %d, want %d", args, code, ExitUsage)
		}
	}
	if code, out, _ := run("year", "--help"); code != ExitOK || !strings.Contains(out, "year [--user U]") {
		t.Errorf("--help: code %d", code)
	}
}

func TestAgo(t *testing.T) {
	for d, want := range map[time.Duration]string{
		10 * time.Second: "just now",
		time.Minute:      "1 minute ago",
		45 * time.Minute: "45 minutes ago",
		3 * time.Hour:    "3 hours ago",
		47 * time.Hour:   "47 hours ago",
		72 * time.Hour:   "3 days ago",
	} {
		if got := ago(d); got != want {
			t.Errorf("ago(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestYearWithoutADatabase(t *testing.T) {
	t.Setenv("TXOKO_DB", "/dev/null/txoko.db") // can't be created
	if code, _, errOut := run("year", "--local"); code != ExitError || errOut == "" {
		t.Errorf("code %d, stderr %q", code, errOut)
	}
}
