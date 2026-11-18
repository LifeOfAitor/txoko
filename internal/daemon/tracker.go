// Package daemon is txoko's background service: once a minute it checks
// whether you're at the computer and, if so, records that minute, so time
// at your desk grows the scene too, not only commits. It also installs
// itself to start when you log in (launchd on macOS, systemd on Linux).
package daemon

import (
	"context"
	"time"
)

// Defaults for the tracker.
const (
	// Every is how often the tracker looks.
	Every = time.Minute
	// Away is how long without touching the keyboard or the mouse before
	// you count as away: reading or thinking for a minute still counts.
	Away = 2 * time.Minute
)

// Tracker samples the idle time and records the minutes you're active.
type Tracker struct {
	Idle   func() (time.Duration, error)
	Record func(ctx context.Context, hour time.Time, minutes float64) error
	Now    func() time.Time
	Every  time.Duration // 0 means Every
	Away   time.Duration // 0 means Away
}

// Tick looks once: if you touched the keyboard or the mouse recently, it
// records one interval of activity in the current hour and reports true.
func (t Tracker) Tick(ctx context.Context) (active bool, err error) {
	idle, err := t.Idle()
	if err != nil {
		return false, err
	}
	if idle >= or(t.Away, Away) {
		return false, nil
	}
	hour := t.Now().Truncate(time.Hour)
	if err := t.Record(ctx, hour, or(t.Every, Every).Minutes()); err != nil {
		return false, err
	}
	return true, nil
}

// Run ticks every interval until ctx is done. A failed tick is reported
// through onError and the tracker keeps going: a busy database or a
// hiccup in the system shouldn't stop it for good. While the computer
// sleeps nothing ticks, so sleep never counts as activity.
func (t Tracker) Run(ctx context.Context, onError func(error)) {
	tick := time.NewTicker(or(t.Every, Every))
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			// A tick cut short because we're stopping isn't worth reporting.
			if _, err := t.Tick(ctx); err != nil && onError != nil && ctx.Err() == nil {
				onError(err)
			}
		}
	}
}

func or(d, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}
