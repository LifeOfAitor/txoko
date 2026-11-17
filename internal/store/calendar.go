package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LifeOfAitor/txoko/internal/contrib"
)

// SaveCalendar keeps a contribution calendar under key, replacing whatever
// was saved there before.
func (s *Store) SaveCalendar(ctx context.Context, key string, c contrib.Calendar) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // a no-op once committed

	if _, err := tx.ExecContext(ctx,
		"INSERT OR REPLACE INTO calendars (key, source, fetched_at) VALUES (?, ?, ?)",
		key, c.Source, c.Fetched.Unix()); err != nil {
		return fmt.Errorf("saving calendar: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM calendar_days WHERE key = ?", key); err != nil {
		return fmt.Errorf("saving calendar: %w", err)
	}
	for _, d := range c.Days {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO calendar_days (key, day, count) VALUES (?, ?, ?)",
			key, d.Date.Format(time.DateOnly), d.Count); err != nil {
			return fmt.Errorf("saving calendar: %w", err)
		}
	}
	return tx.Commit()
}

// LoadCalendar returns the calendar saved under key; ok is false if there
// is none.
func (s *Store) LoadCalendar(ctx context.Context, key string) (c contrib.Calendar, ok bool, err error) {
	var fetched int64
	err = s.db.QueryRowContext(ctx, "SELECT source, fetched_at FROM calendars WHERE key = ?", key).
		Scan(&c.Source, &fetched)
	if errors.Is(err, sql.ErrNoRows) {
		return contrib.Calendar{}, false, nil
	}
	if err != nil {
		return contrib.Calendar{}, false, fmt.Errorf("reading calendar: %w", err)
	}
	c.Fetched = time.Unix(fetched, 0)

	rows, err := s.db.QueryContext(ctx, "SELECT day, count FROM calendar_days WHERE key = ? ORDER BY day", key)
	if err != nil {
		return contrib.Calendar{}, false, fmt.Errorf("reading calendar: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			day string
			d   contrib.Day
		)
		if err := rows.Scan(&day, &d.Count); err != nil {
			return contrib.Calendar{}, false, err
		}
		if d.Date, err = time.Parse(time.DateOnly, day); err != nil {
			return contrib.Calendar{}, false, fmt.Errorf("reading calendar: bad day %q", day)
		}
		c.Days = append(c.Days, d)
	}
	return c, true, rows.Err()
}
