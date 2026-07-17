package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/doxazo-net/canticle/internal/models"
)

// seedDeferredRow enqueues a work_queue row, claims it (Dequeue), then defers
// it (the standard path other queue tests use to reach StatusDeferred; see
// TestDBQueue_NoResultRequeueIsDeferredButReprocessable), returning its id.
func seedDeferredRow(t *testing.T, q *DBQueue, artist, title, sourcePath string) int64 {
	t.Helper()
	ctx := context.Background()
	item, err := q.Enqueue(ctx, models.Inputs{
		Track:      models.Track{ArtistName: artist, TrackName: title},
		Outdir:     "out",
		Filename:   "a.lrc",
		SourcePath: sourcePath,
	}, PriorityScan)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(ctx); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if _, err := q.Defer(ctx, item.ID, time.Hour, errors.New("no results found")); err != nil {
		t.Fatalf("defer: %v", err)
	}
	return item.ID
}

func TestListVocalGateRejections(t *testing.T) {
	q := NewDBQueue(openQueueTestDB(t))
	ctx := context.Background()
	// Seed one deferred row, stamp it not-instrumental with telemetry.
	id := seedDeferredRow(t, q, "Artist", "Title", "/music/a.flac")
	if _, err := q.StampUnclassifiedMiss(ctx, id, InstrumentalTelemetry{MusicSum: 0.97, VocalPeak: 0.04, SpeechMean: 0.001, VocalClass: "Singing", DetectorVersion: "1.17.0"}); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	got, err := q.ListVocalGateRejections(ctx, ListVocalGateRejectionsOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Tel.VocalPeak != 0.04 || got[0].ID != id {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestResetInstrumentalToUnclassified(t *testing.T) {
	q := NewDBQueue(openQueueTestDB(t))
	ctx := context.Background()
	id := seedDeferredRow(t, q, "Artist", "Title", "/music/a.flac")
	if _, err := q.StampUnclassifiedMiss(ctx, id, InstrumentalTelemetry{MusicSum: 0.97, VocalPeak: 0.04, SpeechMean: 0.001, VocalClass: "Singing", DetectorVersion: "1.17.0"}); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	if reset, err := q.ResetInstrumentalToUnclassified(ctx, id); err != nil || !reset {
		t.Fatalf("reset: got %v, %v", reset, err)
	}
}
