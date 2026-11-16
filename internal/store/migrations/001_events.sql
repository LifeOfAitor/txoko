-- Every piece of activity txoko knows about, one row per event.
CREATE TABLE events (
    id   INTEGER PRIMARY KEY,
    kind TEXT    NOT NULL,  -- what happened, e.g. 'commit'
    repo TEXT    NOT NULL,  -- repository name (not its full path)
    ref  TEXT    NOT NULL,  -- unique id of the event, e.g. the commit hash
    at   INTEGER NOT NULL   -- when it happened, in Unix seconds
);

-- The same commit is never counted twice, even if a hook fires again.
CREATE UNIQUE INDEX events_kind_ref ON events (kind, ref);

-- Most reads are "everything since a date".
CREATE INDEX events_at ON events (at);
