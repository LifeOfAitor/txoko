package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LifeOfAitor/txoko/internal/progress"
)

func newLive(src func() (progress.Summary, error)) Model {
	plain := PlainTheme()
	return New(Options{Seed: 42, Now: noon, Theme: &plain, Source: src, Refresh: time.Millisecond})
}

func fixed(s progress.Summary) func() (progress.Summary, error) {
	return func() (progress.Summary, error) { return s, nil }
}

func TestLiveStartsFromSeedAndEasesIn(t *testing.T) {
	m := newLive(fixed(progress.Summary{Growth: 0.8, Today: 3, Streak: 5, Recent: 40}))
	if m.growth != 0 {
		t.Fatalf("live mode should start at 0, got %v", m.growth)
	}

	m, cmd := send(m, m.load())
	if m.target != 0.8 || cmd == nil {
		t.Fatalf("after load: target %v, refresh scheduled %v", m.target, cmd != nil)
	}

	prev := m.growth
	for range 5 {
		m, _ = send(m, tickMsg{id: m.tickID})
		if m.growth <= prev {
			t.Fatalf("growth should rise toward the target: %v -> %v", prev, m.growth)
		}
		prev = m.growth
	}
	for range 200 {
		m, _ = send(m, tickMsg{id: m.tickID})
	}
	if m.growth != 0.8 {
		t.Errorf("growth should settle exactly on the target, got %v", m.growth)
	}
}

func TestLiveRefreshReadsSourceAgain(t *testing.T) {
	calls := 0
	m := newLive(func() (progress.Summary, error) {
		calls++
		return progress.Summary{Growth: float64(calls) / 10}, nil
	})
	m, cmd := send(m, m.load())
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
	m, _ = send(m, cmd()) // the scheduled refresh
	if calls != 2 || m.target != 0.2 {
		t.Errorf("refresh: calls %d, target %v", calls, m.target)
	}
}

func TestLiveIgnoresGrowthKeys(t *testing.T) {
	m := newLive(fixed(progress.Summary{Growth: 0.5}))
	m, _ = send(m, m.load(), key("+"), key("+"), key("-"))
	if m.growth != 0 || m.target != 0.5 {
		t.Errorf("+/- must not change live growth: growth %v target %v", m.growth, m.target)
	}
}

func TestLiveFooter(t *testing.T) {
	tests := []struct {
		name string
		src  func() (progress.Summary, error)
		want string
	}{
		{"activity", fixed(progress.Summary{Growth: 0.6, Today: 3, Streak: 5, Recent: 9}), "today 3 · streak 5d"},
		{"no commits", fixed(progress.Summary{}), "no commits yet"},
		{"error", func() (progress.Summary, error) { return progress.Summary{}, errors.New("disk on fire") }, "can't read activity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newLive(tt.src)
			m, _ = send(m, size(80, 20), m.load())
			view := m.View().Content
			if !strings.Contains(view, tt.want) {
				t.Errorf("footer should contain %q:\n%s", tt.want, lastLines(view, 2))
			}
			if strings.Contains(view, "demo") || strings.Contains(view, "+/-") {
				t.Errorf("live footer must not mention demo controls:\n%s", lastLines(view, 2))
			}
		})
	}
}

func TestLiveErrorKeepsLastGoodData(t *testing.T) {
	fail := false
	m := newLive(func() (progress.Summary, error) {
		if fail {
			return progress.Summary{}, errors.New("locked")
		}
		return progress.Summary{Growth: 0.7}, nil
	})
	m, _ = send(m, m.load())
	fail = true
	m, cmd := send(m, m.load())
	if m.target != 0.7 || m.loadErr == nil || cmd == nil {
		t.Errorf("a failed read should keep the last target and retry: target %v err %v", m.target, m.loadErr)
	}
}

func TestLiveInitLoads(t *testing.T) {
	if newLive(fixed(progress.Summary{})).Init() == nil {
		t.Error("Init should start ticking and loading")
	}
	m := newLive(fixed(progress.Summary{}))
	if _, ok := m.Init()().(tea.BatchMsg); !ok {
		t.Error("live Init should batch the tick and the first load")
	}
}

func TestLiveViewGolden(t *testing.T) {
	m := newLive(fixed(progress.Summary{Growth: 0.9, Today: 4, Streak: 12, Recent: 80}))
	m, _ = send(m, size(80, 20), m.load())
	for range 200 {
		m, _ = send(m, tickMsg{id: m.tickID})
	}
	assertGolden(t, "view_live", m.View().Content)
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}
