// Package store keeps txoko's activity history in a local SQLite database.
//
// Nothing leaves your machine: the database is a single file, by default
// ~/.local/share/txoko/txoko.db. The schema evolves through numbered SQL
// migrations embedded in the binary, applied automatically on Open.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver: no C compiler needed
)

// Kind is the type of an event.
type Kind string

// Kinds of events txoko records.
const (
	KindCommit Kind = "commit"
	KindFocus  Kind = "focus"  // a completed pomodoro session
	KindActive Kind = "active" // minutes at the computer within one hour
)

// Event is one piece of activity.
type Event struct {
	Kind   Kind
	Repo   string    // repository name, or a label such as "pomodoro"
	Ref    string    // unique id within its kind, e.g. a commit hash
	At     time.Time // when it happened
	Amount float64   // size in the kind's unit (focus: minutes); 0 means 1
}

// Store is an open activity database.
type Store struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var migrations embed.FS

// DefaultPath returns where the database lives: $TXOKO_DB if set, otherwise
// $XDG_DATA_HOME/txoko/txoko.db, falling back to ~/.local/share.
func DefaultPath() (string, error) {
	if p := os.Getenv("TXOKO_DB"); p != "" {
		return p, nil
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "txoko", "txoko.db"), nil
}

// Open opens (creating if needed) the database at path and brings its
// schema up to date.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	// WAL lets the TUI read while a git hook writes; busy_timeout makes a
	// writer wait briefly instead of failing if the database is locked.
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// migrate applies every embedded migration newer than the database's
// schema version, which SQLite stores in PRAGMA user_version.
func (s *Store) migrate(ctx context.Context) error {
	var current int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files) // 001_..., 002_..., applied in order

	for i, name := range files {
		version := i + 1
		if version <= current {
			continue
		}
		body, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		// Each migration and its version bump succeed or fail together.
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

// ErrInvalidEvent is returned when an event is missing required fields.
var ErrInvalidEvent = errors.New("event needs a kind, repo, ref and time")

// Add records an event. It reports false when the event was already known
// (same kind and ref), which is not an error.
func (s *Store) Add(ctx context.Context, e Event) (bool, error) {
	if e.Kind == "" || e.Repo == "" || e.Ref == "" || e.At.IsZero() {
		return false, ErrInvalidEvent
	}
	if e.Amount == 0 {
		e.Amount = 1
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO events (kind, repo, ref, at, amount) VALUES (?, ?, ?, ?, ?)",
		string(e.Kind), e.Repo, e.Ref, e.At.Unix(), e.Amount)
	if err != nil {
		return false, fmt.Errorf("adding event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// AddActive adds minutes at the computer to the hour starting at hour.
// Each hour is a single event whose amount grows as the minutes come in,
// so a day at the desk is a handful of rows, not hundreds.
func (s *Store) AddActive(ctx context.Context, hour time.Time, minutes float64) error {
	if hour.IsZero() || minutes <= 0 {
		return ErrInvalidEvent
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (kind, repo, ref, at, amount) VALUES (?, 'computer', ?, ?, ?)
		ON CONFLICT (kind, ref) DO UPDATE SET amount = amount + excluded.amount`,
		string(KindActive), hour.UTC().Format("2006-01-02T15"), hour.Unix(), minutes)
	if err != nil {
		return fmt.Errorf("adding active time: %w", err)
	}
	return nil
}

// Since returns every event at or after t, oldest first.
func (s *Store) Since(ctx context.Context, t time.Time) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT kind, repo, ref, at, amount FROM events WHERE at >= ? ORDER BY at, id", t.Unix())
	if err != nil {
		return nil, fmt.Errorf("reading events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []Event
	for rows.Next() {
		var (
			e    Event
			kind string
			at   int64
		)
		if err := rows.Scan(&kind, &e.Repo, &e.Ref, &at, &e.Amount); err != nil {
			return nil, err
		}
		e.Kind, e.At = Kind(kind), time.Unix(at, 0)
		events = append(events, e)
	}
	return events, rows.Err()
}

// Count returns how many events are stored.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&n)
	return n, err
}
