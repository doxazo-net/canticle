-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
-- Replace the (artist, title, album) unique key with (artist, title, duration_bucket).
-- Album is ignored by the Musixmatch API so it was a useless differentiator;
-- the same recording stored under two album tags created two cache rows with
-- identical lyrics. duration_bucket = floor(seconds/5) with sentinel 0 for
-- unknown duration, which degrades the key to (artist, title) until #191 wires
-- in real duration data.
--
-- Uses the explicit-transaction + FK-toggle pattern from 012: PRAGMA foreign_keys
-- must be toggled OUTSIDE a transaction (it is a no-op inside one), so this
-- migration is NO TRANSACTION and manages BEGIN/COMMIT itself. lyrics_cache has
-- no outbound FK references, but the toggle prevents DROP from cascading through
-- any future inbound references.
--
-- Crash safety: the DROP + RENAME runs inside an explicit transaction so a crash
-- mid-migration rolls back cleanly, leaving lyrics_cache intact and goose's
-- version table unchanged; the next startup re-runs from a consistent state.

PRAGMA foreign_keys = OFF;
DROP TABLE IF EXISTS lyrics_cache_new;

BEGIN;

CREATE TABLE lyrics_cache_new (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    artist          TEXT     NOT NULL,
    title           TEXT     NOT NULL,
    duration_bucket INTEGER  NOT NULL DEFAULT 0,
    lyrics          TEXT     NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(artist, title, duration_bucket)
);

-- Dedup existing rows: keep the most-recently-updated row per (artist, title)
-- and migrate all to duration_bucket=0 (unknown-duration sentinel).
INSERT INTO lyrics_cache_new (artist, title, duration_bucket, lyrics, created_at, updated_at)
WITH ranked AS (
    SELECT artist, title, lyrics, created_at, updated_at,
           ROW_NUMBER() OVER (
               PARTITION BY artist, title
               ORDER BY updated_at DESC, id DESC
           ) AS rn
    FROM lyrics_cache
)
SELECT artist, title, 0, lyrics, created_at, updated_at
FROM ranked
WHERE rn = 1;

DROP TRIGGER IF EXISTS update_lyrics_cache_updated_at;
DROP TABLE lyrics_cache;
ALTER TABLE lyrics_cache_new RENAME TO lyrics_cache;

CREATE TRIGGER IF NOT EXISTS update_lyrics_cache_updated_at
AFTER UPDATE ON lyrics_cache
BEGIN
    UPDATE lyrics_cache SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
    WHERE id = NEW.id;
END;

COMMIT;

PRAGMA foreign_keys = ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse: restore UNIQUE(artist, title, album) with album TEXT DEFAULT ''.
-- Note: the original album values are LOST on rollback - they are not stored
-- anywhere after the Up migration runs. This is acceptable because album content
-- survives in the Song JSON blobs stored in the lyrics column; the cache is a
-- performance layer and can be rebuilt. Any cache rows added after Up ran will be
-- deduplicated to album='' here (keeping most-recently-updated per artist+title).

PRAGMA foreign_keys = OFF;
DROP TABLE IF EXISTS lyrics_cache_old;

BEGIN;

CREATE TABLE lyrics_cache_old (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    artist     TEXT     NOT NULL,
    title      TEXT     NOT NULL,
    album      TEXT     NOT NULL DEFAULT '',
    lyrics     TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(artist, title, album)
);

INSERT INTO lyrics_cache_old (artist, title, album, lyrics, created_at, updated_at)
WITH ranked AS (
    SELECT artist, title, lyrics, created_at, updated_at,
           ROW_NUMBER() OVER (
               PARTITION BY artist, title
               ORDER BY updated_at DESC, id DESC
           ) AS rn,
           MIN(created_at) OVER (PARTITION BY artist, title) AS earliest_created_at
    FROM lyrics_cache
)
SELECT artist, title, '', lyrics, earliest_created_at, updated_at
FROM ranked
WHERE rn = 1;

DROP TRIGGER IF EXISTS update_lyrics_cache_updated_at;
DROP TABLE lyrics_cache;
ALTER TABLE lyrics_cache_old RENAME TO lyrics_cache;

CREATE TRIGGER IF NOT EXISTS update_lyrics_cache_updated_at
AFTER UPDATE ON lyrics_cache
BEGIN
    UPDATE lyrics_cache SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
    WHERE id = NEW.id;
END;

COMMIT;

PRAGMA foreign_keys = ON;
-- +goose StatementEnd
