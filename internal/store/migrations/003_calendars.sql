-- Contribution calendars read from elsewhere (GitHub), kept so txoko
-- doesn't ask again on every run and still works offline.
CREATE TABLE calendars (
    key        TEXT PRIMARY KEY,  -- which calendar, e.g. 'github:@me'
    source     TEXT    NOT NULL,  -- where it came from, for display
    fetched_at INTEGER NOT NULL   -- when it was read, in Unix seconds
);

-- One row per day of each calendar.
CREATE TABLE calendar_days (
    key   TEXT    NOT NULL,  -- the calendar it belongs to
    day   TEXT    NOT NULL,  -- the calendar day, as 2006-01-02
    count INTEGER NOT NULL,
    PRIMARY KEY (key, day)
);
