package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sydlexius/mxlrcgo-svc/internal/normalize"
)

// CacheRepo provides read/write access to the lyrics_cache table.
// All artist/title strings are normalized before storage and lookup.
// The unique cache key is (artist, title, duration_bucket); bucket 0 is the
// unknown-duration sentinel (see #191 for real-duration wiring).
type CacheRepo struct {
	db *sql.DB
}

// New returns a CacheRepo backed by db.
func New(db *sql.DB) *CacheRepo {
	return &CacheRepo{db: db}
}

// Lookup returns the cached lyrics for (artist, title, durationBucket) after
// normalization. Pass durationBucket=0 when the recording duration is unknown.
// Returns sql.ErrNoRows if not found.
func (r *CacheRepo) Lookup(ctx context.Context, artist, title string, durationBucket int) (string, error) {
	var lyrics string
	err := r.db.QueryRowContext(ctx,
		`SELECT lyrics FROM lyrics_cache WHERE artist=? AND title=? AND duration_bucket=? LIMIT 1`,
		normalize.NormalizeKey(artist),
		normalize.NormalizeKey(title),
		durationBucket,
	).Scan(&lyrics)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("cache: lookup: %w", err)
	}
	return lyrics, nil
}

// Store inserts or updates (upsert) the lyrics for (artist, title, durationBucket).
// Keys are normalized before storage. updated_at is maintained by a database trigger.
func (r *CacheRepo) Store(ctx context.Context, artist, title string, durationBucket int, lyrics string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO lyrics_cache (artist, title, duration_bucket, lyrics)
         VALUES (?, ?, ?, ?)
         ON CONFLICT(artist, title, duration_bucket) DO UPDATE SET
             lyrics = excluded.lyrics`,
		normalize.NormalizeKey(artist),
		normalize.NormalizeKey(title),
		durationBucket,
		lyrics,
	)
	if err != nil {
		return fmt.Errorf("cache: store: %w", err)
	}
	return nil
}
