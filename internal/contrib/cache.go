package contrib

import (
	"context"
	"time"
)

// MaxAge is how long a calendar read from GitHub is reused before asking
// again. Contributions change slowly, and GitHub limits how often it may
// be asked.
const MaxAge = 6 * time.Hour

// Cache keeps calendars between runs, under a key such as "github:@me".
type Cache interface {
	LoadCalendar(ctx context.Context, key string) (c Calendar, ok bool, err error)
	SaveCalendar(ctx context.Context, key string, c Calendar) error
}

// Cached returns the calendar saved under key while it's younger than
// MaxAge, unless refresh is set. Otherwise it fetches a new one, stamps it
// with now and saves it. When fetching fails but an older calendar was
// saved, that one comes back anyway, with the failure as a warning: an old
// calendar beats none when you're offline.
func Cached(ctx context.Context, cache Cache, key string, now time.Time, refresh bool,
	fetch func(context.Context) (Calendar, error),
) (c Calendar, warning, err error) {
	saved, ok, err := cache.LoadCalendar(ctx, key)
	if err != nil {
		return Calendar{}, nil, err
	}
	if ok && !refresh && now.Sub(saved.Fetched) < MaxAge {
		return saved, nil, nil
	}
	fresh, err := fetch(ctx)
	if err != nil {
		if ok {
			return saved, err, nil
		}
		return Calendar{}, nil, err
	}
	fresh.Fetched = now
	if err := cache.SaveCalendar(ctx, key, fresh); err != nil {
		return Calendar{}, nil, err
	}
	return fresh, nil, nil
}
