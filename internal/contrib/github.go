package contrib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Endpoint is GitHub's GraphQL API.
const Endpoint = "https://api.github.com/graphql"

// Errors from GitHub that deserve their own message.
var (
	ErrNoToken      = errors.New("no GitHub token: set GITHUB_TOKEN, or log in with `gh auth login`")
	ErrUnauthorized = errors.New("GitHub refused the token: it may have expired")
	ErrNoUser       = errors.New("no such GitHub user")
)

// GitHub reads contribution calendars from GitHub's GraphQL API. It only
// ever reads: txoko sends nothing about you to GitHub.
type GitHub struct {
	Endpoint string       // the API; Endpoint unless testing
	Token    string       // any token that can read public profiles
	HTTP     *http.Client // nil means http.DefaultClient
}

// The query asks for one thing: the calendar GitHub draws on a profile,
// for the last year. With no login it's the token owner's ("viewer").
const (
	calendarFields = `login contributionsCollection { contributionCalendar {
		weeks { contributionDays { date contributionCount } } } }`
	viewerQuery = `query { viewer { ` + calendarFields + ` } }`
	userQuery   = `query($login: String!) { user(login: $login) { ` + calendarFields + ` } }`
)

// graphQL is a request to the API.
type graphQL struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables,omitempty"`
}

type calendarUser struct {
	Login                   string
	ContributionsCollection struct {
		ContributionCalendar struct {
			Weeks []struct {
				ContributionDays []struct {
					Date              string
					ContributionCount int
				}
			}
		}
	}
}

// Calendar fetches the last year of contributions of login, or of the
// token's owner when login is empty.
func (g GitHub) Calendar(ctx context.Context, login string) (Calendar, error) {
	body := graphQL{Query: viewerQuery}
	if login != "" {
		body = graphQL{Query: userQuery, Variables: map[string]string{"login": login}}
	}
	payload, _ := json.Marshal(body) // only strings: it always encodes
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return Calendar{}, err
	}
	req.Header.Set("Authorization", "bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")

	client := g.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Calendar{}, fmt.Errorf("reaching GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return Calendar{}, ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		return Calendar{}, fmt.Errorf("GitHub answered %s", resp.Status)
	}

	var out struct {
		Data struct {
			Viewer *calendarUser
			User   *calendarUser
		}
		Errors []struct{ Type, Message string }
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return Calendar{}, fmt.Errorf("reading GitHub's answer: %w", err)
	}
	if len(out.Errors) > 0 {
		if out.Errors[0].Type == "NOT_FOUND" {
			return Calendar{}, fmt.Errorf("%w: %s", ErrNoUser, login)
		}
		return Calendar{}, fmt.Errorf("GitHub: %s", out.Errors[0].Message)
	}
	user := out.Data.Viewer
	if login != "" {
		user = out.Data.User
	}
	if user == nil {
		return Calendar{}, fmt.Errorf("%w: %s", ErrNoUser, login)
	}

	c := Calendar{Source: "@" + user.Login + " on GitHub"}
	for _, w := range user.ContributionsCollection.ContributionCalendar.Weeks {
		for _, d := range w.ContributionDays {
			date, err := time.Parse(time.DateOnly, d.Date)
			if err != nil {
				return Calendar{}, fmt.Errorf("reading GitHub's answer: bad date %q", d.Date)
			}
			c.Days = append(c.Days, Day{Date: date, Count: d.ContributionCount})
		}
	}
	return c, nil
}

// FindToken looks for a GitHub token the way GitHub's own tools do: in
// GITHUB_TOKEN or GH_TOKEN, then from the GitHub CLI if you're logged in.
// txoko never stores it.
func FindToken(getenv func(string) string, gh func() (string, error)) (string, error) {
	for _, name := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if t := strings.TrimSpace(getenv(name)); t != "" {
			return t, nil
		}
	}
	if t, err := gh(); err == nil && strings.TrimSpace(t) != "" {
		return strings.TrimSpace(t), nil
	}
	return "", ErrNoToken
}

// GHToken asks the GitHub CLI for the token it's logged in with.
func GHToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	return string(out), err
}

// EnvToken finds a token in the environment or the GitHub CLI.
func EnvToken() (string, error) { return FindToken(os.Getenv, GHToken) }
