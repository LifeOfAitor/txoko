package store

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "nested", "txoko.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func commit(ref string, at time.Time) Event {
	return Event{Kind: KindCommit, Repo: "txoko", Ref: ref, At: at}
}

func TestAddAndSince(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	base := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	for i, ref := range []string{"c", "a", "b"} {
		if added, err := s.Add(ctx, commit(ref, base.Add(time.Duration(i)*time.Hour))); err != nil || !added {
			t.Fatalf("Add(%s) = %v, %v", ref, added, err)
		}
	}

	got, err := s.Since(ctx, base.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Ref != "a" || got[1].Ref != "b" {
		t.Fatalf("Since = %+v, want a then b", got)
	}
	if !got[0].At.Equal(base.Add(time.Hour)) || got[0].Kind != KindCommit || got[0].Repo != "txoko" {
		t.Errorf("event round-trip mismatch: %+v", got[0])
	}
}

func TestAddIgnoresDuplicates(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	e := commit("abc123", time.Now())

	if added, _ := s.Add(ctx, e); !added {
		t.Fatal("first Add should insert")
	}
	added, err := s.Add(ctx, e)
	if err != nil || added {
		t.Fatalf("second Add = %v, %v; want false, nil", added, err)
	}
	if n, _ := s.Count(ctx); n != 1 {
		t.Errorf("Count = %d, want 1", n)
	}
}

func TestAddRejectsInvalidEvents(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	now := time.Now()
	for _, e := range []Event{
		{Repo: "r", Ref: "x", At: now},
		{Kind: KindCommit, Ref: "x", At: now},
		{Kind: KindCommit, Repo: "r", At: now},
		{Kind: KindCommit, Repo: "r", Ref: "x"},
	} {
		if _, err := s.Add(ctx, e); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("Add(%+v) error = %v, want ErrInvalidEvent", e, err)
		}
	}
}

func TestReopenKeepsDataAndSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "txoko.db")

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(ctx, commit("keep", time.Now())); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s, err = Open(ctx, path) // migrations must not run twice
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if n, _ := s.Count(ctx); n != 1 {
		t.Errorf("Count after reopen = %d, want 1", n)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Errorf("user_version = %d, %v; want 1", version, err)
	}
}

// TestConcurrentWriters mimics several git hooks firing at once.
func TestConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "txoko.db")
	if s, err := Open(ctx, path); err != nil {
		t.Fatal(err)
	} else {
		_ = s.Close()
	}

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := range 8 {
		wg.Go(func() {
			s, err := Open(ctx, path)
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = s.Close() }()
			_, err = s.Add(ctx, commit(string(rune('a'+i)), time.Now()))
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}

	check, _ := Open(ctx, path)
	defer func() { _ = check.Close() }()
	if n, _ := check.Count(ctx); n != 8 {
		t.Errorf("Count = %d, want 8", n)
	}
}

func TestOpenFailsOnBadPath(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	s, err := Open(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	// A path below a regular file can't be created.
	if _, err := Open(context.Background(), filepath.Join(file, "txoko.db")); err == nil {
		t.Error("Open below a file should fail")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("TXOKO_DB", "/tmp/custom.db")
	if p, _ := DefaultPath(); p != "/tmp/custom.db" {
		t.Errorf("TXOKO_DB not honored: %s", p)
	}

	t.Setenv("TXOKO_DB", "")
	t.Setenv("XDG_DATA_HOME", "/data")
	if p, _ := DefaultPath(); p != "/data/txoko/txoko.db" {
		t.Errorf("XDG_DATA_HOME not honored: %s", p)
	}

	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/me")
	if p, _ := DefaultPath(); p != "/home/me/.local/share/txoko/txoko.db" {
		t.Errorf("fallback path = %s", p)
	}
}
