package contrib

import (
	"context"
	"errors"
	"testing"
	"time"
)

// memory is a Cache in a map, that can be told to fail.
type memory struct {
	saved            map[string]Calendar
	loadErr, saveErr error
}

func (m *memory) LoadCalendar(_ context.Context, key string) (Calendar, bool, error) {
	c, ok := m.saved[key]
	return c, ok, m.loadErr
}

func (m *memory) SaveCalendar(_ context.Context, key string, c Calendar) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved[key] = c
	return nil
}

func TestCached(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 11, 17, 12, 0, 0, 0, time.UTC)
	old := Calendar{Source: "old", Fetched: now.Add(-MaxAge - time.Minute)}
	recent := Calendar{Source: "recent", Fetched: now.Add(-time.Hour)}
	fetched := Calendar{Source: "new"}
	offline := errors.New("offline")

	for _, tt := range []struct {
		name     string
		saved    *Calendar
		refresh  bool
		fetchErr error
		want     string
		fetches  bool
		warning  bool
		wantErr  bool
	}{
		{"nothing saved: fetch", nil, false, nil, "new", true, false, false},
		{"recent: reuse it", &recent, false, nil, "recent", false, false, false},
		{"recent but refresh: fetch", &recent, true, nil, "new", true, false, false},
		{"old: fetch", &old, false, nil, "new", true, false, false},
		{"old and offline: old, with a warning", &old, false, offline, "old", true, true, false},
		{"nothing saved and offline: error", nil, false, offline, "", true, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cache := &memory{saved: map[string]Calendar{}}
			if tt.saved != nil {
				cache.saved["k"] = *tt.saved
			}
			fetches := false
			c, warning, err := Cached(ctx, cache, "k", now, tt.refresh, func(context.Context) (Calendar, error) {
				fetches = true
				return fetched, tt.fetchErr
			})
			if c.Source != tt.want || fetches != tt.fetches || (warning != nil) != tt.warning || (err != nil) != tt.wantErr {
				t.Errorf("got %q, fetched %v, warning %v, err %v", c.Source, fetches, warning, err)
			}
			if tt.want == "new" && (!cache.saved["k"].Fetched.Equal(now) || c.Fetched != now) {
				t.Error("a new calendar is stamped with now and saved")
			}
		})
	}
}

func TestCachedStorageErrors(t *testing.T) {
	ctx := context.Background()
	fetch := func(context.Context) (Calendar, error) { return Calendar{}, nil }
	broken := errors.New("disk on fire")
	if _, _, err := Cached(ctx, &memory{loadErr: broken}, "k", time.Now(), false, fetch); !errors.Is(err, broken) {
		t.Errorf("load error: %v", err)
	}
	if _, _, err := Cached(ctx, &memory{saved: map[string]Calendar{}, saveErr: broken}, "k", time.Now(), false, fetch); !errors.Is(err, broken) {
		t.Errorf("save error: %v", err)
	}
}
