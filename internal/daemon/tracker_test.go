package daemon

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type recorded struct {
	hour    time.Time
	minutes float64
}

// recorder collects what a tracker records, and can be told to fail.
type recorder struct {
	mu   sync.Mutex
	got  []recorded
	fail error
}

func (r *recorder) record(_ context.Context, hour time.Time, minutes float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	r.got = append(r.got, recorded{hour, minutes})
	return nil
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.got)
}

func idleFor(d time.Duration, err error) func() (time.Duration, error) {
	return func() (time.Duration, error) { return d, err }
}

func TestTick(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 16, 42, 10, 0, time.UTC)
	clock := func() time.Time { return at }

	r := &recorder{}
	active, err := Tracker{Idle: idleFor(30*time.Second, nil), Record: r.record, Now: clock}.Tick(ctx)
	if !active || err != nil || len(r.got) != 1 {
		t.Fatalf("active=%v err=%v got %v", active, err, r.got)
	}
	if want := time.Date(2026, 11, 18, 16, 0, 0, 0, time.UTC); !r.got[0].hour.Equal(want) || r.got[0].minutes != 1 {
		t.Errorf("recorded %v, want one minute in the hour of %v", r.got[0], want)
	}

	// Away long enough: nothing is recorded.
	r = &recorder{}
	if active, err := (Tracker{Idle: idleFor(Away, nil), Record: r.record, Now: clock}).Tick(ctx); active || err != nil || len(r.got) != 0 {
		t.Errorf("away: active=%v err=%v got %v", active, err, r.got)
	}

	// Custom interval and threshold.
	r = &recorder{}
	tr := Tracker{Idle: idleFor(4*time.Minute, nil), Record: r.record, Now: clock, Every: 5 * time.Minute, Away: 10 * time.Minute}
	if active, _ := tr.Tick(ctx); !active || r.got[0].minutes != 5 {
		t.Errorf("custom: active=%v got %v", active, r.got)
	}
}

func TestTickErrors(t *testing.T) {
	ctx := context.Background()
	noIdle := errors.New("no idea")
	if _, err := (Tracker{Idle: idleFor(0, noIdle)}).Tick(ctx); !errors.Is(err, noIdle) {
		t.Errorf("idle error: %v", err)
	}
	full := errors.New("disk full")
	r := &recorder{fail: full}
	if _, err := (Tracker{Idle: idleFor(0, nil), Record: r.record, Now: time.Now}).Tick(ctx); !errors.Is(err, full) {
		t.Errorf("record error: %v", err)
	}
}

// TestRun ticks fast, reports failures without stopping, and stops when
// asked.
func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &recorder{}
	var errs []error
	var mu sync.Mutex
	calls := 0
	idle := func() (time.Duration, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == 2 {
			return 0, errors.New("a hiccup")
		}
		return 0, nil
	}
	done := make(chan struct{})
	go func() {
		Tracker{Idle: idle, Record: r.record, Now: time.Now, Every: time.Millisecond}.Run(ctx, func(err error) {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		})
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for r.count() < 3 {
		select {
		case <-deadline:
			t.Fatal("the tracker stopped ticking")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	<-done
	mu.Lock()
	defer mu.Unlock()
	if len(errs) != 1 {
		t.Errorf("one hiccup reported, got %v", errs)
	}

	// Without an error handler, failures are simply skipped.
	ctx, cancel = context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	Tracker{Idle: idleFor(0, errors.New("x")), Every: time.Millisecond}.Run(ctx, nil)
}

// TestRunQuietOnStop: a tick that fails because the tracker is stopping
// isn't reported.
func TestRunQuietOnStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var reported []error
	record := func(context.Context, time.Time, float64) error {
		cancel() // the stop arrives while this tick writes
		return context.Canceled
	}
	Tracker{Idle: idleFor(0, nil), Record: record, Now: time.Now, Every: time.Millisecond}.Run(ctx, func(err error) {
		reported = append(reported, err)
	})
	if len(reported) != 0 {
		t.Errorf("reported %v while stopping", reported)
	}
}
