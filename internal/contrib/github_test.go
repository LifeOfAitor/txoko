package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitHub answers GraphQL requests like GitHub would, checking that they
// carry the token.
func fakeGitHub(t *testing.T, status int, answer string) (*httptest.Server, *map[string]any) {
	t.Helper()
	got := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "bearer secret" {
			t.Errorf("request %s with auth %q", r.Method, r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

const calendarJSON = `{"data": {"%s": {"login": "octo", "contributionsCollection": {"contributionCalendar": {"weeks": [
	{"contributionDays": [{"date": "2025-11-02", "contributionCount": 0}, {"date": "2025-11-03", "contributionCount": 4}]},
	{"contributionDays": [{"date": "2025-11-09", "contributionCount": 2}]}
]}}}}}`

func TestCalendarOfTheViewer(t *testing.T) {
	srv, req := fakeGitHub(t, http.StatusOK, strings.Replace(calendarJSON, "%s", "viewer", 1))
	c, err := GitHub{Endpoint: srv.URL, Token: "secret"}.Calendar(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if q, _ := (*req)["query"].(string); !strings.Contains(q, "viewer") || (*req)["variables"] != nil {
		t.Errorf("asked %v", *req)
	}
	if c.Source != "@octo on GitHub" || len(c.Days) != 3 || c.Total() != 6 {
		t.Errorf("got %+v", c)
	}
	if c.Days[1].Date.Format("2006-01-02") != "2025-11-03" || c.Days[1].Count != 4 {
		t.Errorf("second day = %+v", c.Days[1])
	}
}

func TestCalendarOfAUser(t *testing.T) {
	srv, req := fakeGitHub(t, http.StatusOK, strings.Replace(calendarJSON, "%s", "user", 1))
	c, err := GitHub{Endpoint: srv.URL, Token: "secret", HTTP: srv.Client()}.Calendar(context.Background(), "octo")
	if err != nil {
		t.Fatal(err)
	}
	vars, _ := (*req)["variables"].(map[string]any)
	if q, _ := (*req)["query"].(string); !strings.Contains(q, "user(login") || vars["login"] != "octo" {
		t.Errorf("asked %v", *req)
	}
	if c.Total() != 6 {
		t.Errorf("got %+v", c)
	}
}

func TestCalendarErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		answer string
		login  string
		want   error  // errors.Is, or nil to only check the text
		text   string // part of the message
	}{
		{"bad token", http.StatusUnauthorized, `{}`, "", ErrUnauthorized, ""},
		{"server trouble", http.StatusBadGateway, ``, "", nil, "502"},
		{"no such user", http.StatusOK, `{"data": {"user": null}, "errors": [{"type": "NOT_FOUND", "message": "nope"}]}`, "ghost", ErrNoUser, "ghost"},
		{"user missing silently", http.StatusOK, `{"data": {"user": null}}`, "ghost", ErrNoUser, ""},
		{"other GraphQL error", http.StatusOK, `{"errors": [{"type": "RATE_LIMITED", "message": "slow down"}]}`, "", nil, "slow down"},
		{"not JSON", http.StatusOK, `<html>`, "", nil, "reading GitHub's answer"},
		{"bad date", http.StatusOK, strings.Replace(strings.Replace(calendarJSON, "%s", "viewer", 1), "2025-11-03", "soon", 1), "", nil, "bad date"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := fakeGitHub(t, tt.status, tt.answer)
			_, err := GitHub{Endpoint: srv.URL, Token: "secret"}.Calendar(context.Background(), tt.login)
			if err == nil || tt.want != nil && !errors.Is(err, tt.want) || !strings.Contains(err.Error(), tt.text) {
				t.Errorf("got %v", err)
			}
		})
	}
}

func TestCalendarUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // nobody is listening any more
	if _, err := (GitHub{Endpoint: srv.URL, Token: "secret"}).Calendar(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "reaching GitHub") {
		t.Errorf("got %v", err)
	}
	if _, err := (GitHub{Endpoint: "://bad", Token: "secret"}).Calendar(context.Background(), ""); err == nil {
		t.Error("a bad endpoint is an error")
	}
}

func TestFindToken(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}
	gh := func(token string, err error) func() (string, error) {
		return func() (string, error) { return token, err }
	}
	for _, tt := range []struct {
		name string
		env  map[string]string
		gh   func() (string, error)
		want string
	}{
		{"GITHUB_TOKEN first", map[string]string{"GITHUB_TOKEN": "a", "GH_TOKEN": "b"}, gh("c", nil), "a"},
		{"then GH_TOKEN", map[string]string{"GITHUB_TOKEN": " ", "GH_TOKEN": "b\n"}, gh("c", nil), "b"},
		{"then the GitHub CLI", nil, gh("c\n", nil), "c"},
		{"nothing", nil, gh("", errors.New("not logged in")), ""},
		{"CLI with nothing to say", nil, gh("  \n", nil), ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindToken(env(tt.env), tt.gh)
			if got != tt.want || (tt.want == "") != errors.Is(err, ErrNoToken) {
				t.Errorf("got %q, %v", got, err)
			}
		})
	}
}

// TestEnvToken uses a fake gh on the PATH, so it never reads a real token.
func TestEnvToken(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho fake-token\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	if got, err := EnvToken(); got != "fake-token" || err != nil {
		t.Errorf("EnvToken() = %q, %v", got, err)
	}
}
