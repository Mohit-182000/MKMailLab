-- Message IDs must never be reused: the UI keys selections, caches and live
-- events by ID, and SQLite's implicit rowid restarts at 1 once the table is
-- empty (e.g. after "Delete all"). A monotonic counter is used instead.
-- (Rebuilding `emails` with AUTOINCREMENT would require DROP TABLE, which
-- cascades deletes to every child table while foreign keys are enforced.)

CREATE TABLE id_sequence (
    name  TEXT PRIMARY KEY,
    value INTEGER NOT NULL
) WITHOUT ROWID;

INSERT INTO id_sequence (name, value)
VALUES ('emails', (SELECT COALESCE(MAX(id), 0) FROM emails));
