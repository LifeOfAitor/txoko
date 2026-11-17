-- How much an event is worth in its own unit: 1 for a commit, the number of
-- minutes for a focus session. Existing rows are commits, so they get 1.
ALTER TABLE events ADD COLUMN amount REAL NOT NULL DEFAULT 1;
