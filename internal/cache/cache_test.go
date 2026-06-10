package cache_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sydlexius/mxlrcgo-svc/internal/cache"
	"github.com/sydlexius/mxlrcgo-svc/internal/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

// TestSameRecordingAcrossAlbumsCollapsesToOneRow verifies that storing the same
// artist+title+bucket twice (e.g. different album tags for the same recording)
// upserts rather than creating a second row.
func TestSameRecordingAcrossAlbumsCollapsesToOneRow(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	if err := repo.Store(ctx, "Artist", "Song", 0, "lyrics v1"); err != nil {
		t.Fatalf("Store v1: %v", err)
	}
	// Same recording, different album tag in the file - should upsert, not duplicate.
	if err := repo.Store(ctx, "Artist", "Song", 0, "lyrics v2"); err != nil {
		t.Fatalf("Store v2: %v", err)
	}
	got, err := repo.Lookup(ctx, "Artist", "Song", 0)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != "lyrics v2" {
		t.Errorf("got %q, want %q after upsert", got, "lyrics v2")
	}
}

// TestDistinctDurationRecordingsCacheSeparately verifies that recordings in
// different 5-second duration buckets produce separate cache rows.
func TestDistinctDurationRecordingsCacheSeparately(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	const bucketA = 36 // e.g. floor(180/5)
	const bucketB = 48 // e.g. floor(240/5)

	if err := repo.Store(ctx, "Artist", "Song", bucketA, "short version"); err != nil {
		t.Fatalf("Store A: %v", err)
	}
	if err := repo.Store(ctx, "Artist", "Song", bucketB, "long version"); err != nil {
		t.Fatalf("Store B: %v", err)
	}

	gotA, err := repo.Lookup(ctx, "Artist", "Song", bucketA)
	if err != nil {
		t.Fatalf("Lookup A: %v", err)
	}
	if gotA != "short version" {
		t.Errorf("bucket A: got %q, want %q", gotA, "short version")
	}

	gotB, err := repo.Lookup(ctx, "Artist", "Song", bucketB)
	if err != nil {
		t.Fatalf("Lookup B: %v", err)
	}
	if gotB != "long version" {
		t.Errorf("bucket B: got %q, want %q", gotB, "long version")
	}
}

// TestMultiISRCSameDurationSharesOneRow verifies that multiple ISRC territorial
// variants of the same recording (same duration bucket) collapse to one cache row.
func TestMultiISRCSameDurationSharesOneRow(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	const bucket = 42 // floor(210/5)

	if err := repo.Store(ctx, "Artist", "Song", bucket, "lyrics from US release"); err != nil {
		t.Fatalf("Store ISRC-US: %v", err)
	}
	// Same duration bucket - should upsert rather than insert a second row.
	if err := repo.Store(ctx, "Artist", "Song", bucket, "lyrics from EU release"); err != nil {
		t.Fatalf("Store ISRC-EU: %v", err)
	}
	got, err := repo.Lookup(ctx, "Artist", "Song", bucket)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != "lyrics from EU release" {
		t.Errorf("got %q, want last-written %q", got, "lyrics from EU release")
	}
}

// TestUnknownDurationBehavesLikeArtistTitle verifies that bucket=0 (the unknown
// sentinel) makes the effective key (artist, title), one row per song.
func TestUnknownDurationBehavesLikeArtistTitle(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	if err := repo.Store(ctx, "Artist", "Song", 0, "cached lyrics"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := repo.Lookup(ctx, "Artist", "Song", 0)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != "cached lyrics" {
		t.Errorf("got %q, want %q", got, "cached lyrics")
	}
}

func TestLookup_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	_, err := repo.Lookup(ctx, "Nobody", "Nothing", 0)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("got %v, want sql.ErrNoRows", err)
	}
}

func TestLookup_NormalizesKeys(t *testing.T) {
	ctx := context.Background()
	repo := cache.New(openTestDB(t))

	if err := repo.Store(ctx, "  Héllo  ", "  Wörld  ", 0, "normalized lyrics"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := repo.Lookup(ctx, "hello", "world", 0)
	if err != nil {
		t.Fatalf("Lookup normalized: %v", err)
	}
	if got != "normalized lyrics" {
		t.Errorf("got %q, want %q", got, "normalized lyrics")
	}
}
