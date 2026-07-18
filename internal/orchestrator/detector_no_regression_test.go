package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/sydlexius/canticle/internal/circuit"
	"github.com/sydlexius/canticle/internal/detector"
	"github.com/sydlexius/canticle/internal/models"
)

// These tests cover #501's acceptance criterion "Behavior in Music (high
// provider lyric coverage) is not regressed by the reordering" (#536).
//
// The existing detector_lane_test.go cases exercise the lane in isolation. What
// was uncovered is the COMPOSED case: a detector lane running FIRST, ahead of a
// provider that can serve the track. That is the configuration a
// high-lyric-coverage library runs in, and the one where a regression would be
// invisible -- it surfaces as tracks quietly no longer getting lyrics, which
// reads as provider flakiness rather than a detector bug.
//
// The property under test is that the detector must not preempt a provider
// unless its verdict is gate-positive.

// vocalTrackDetector returns a gate-negative verdict, which is what the detector
// reports for an ordinary sung track: it contributes no result and the providers
// must run exactly as they did before the lane existed.
func vocalTrackDetector() *stubDetector {
	return &stubDetector{res: detector.Result{
		Instrumental:      false,
		Confidence:        0.10,
		VocalConfidence:   0.85,
		SpeechConfidence:  0.02,
		WinningVocalClass: "Singing",
		Version:           "1.5.0",
	}}
}

// detectorLaneFront builds the lane order a high-lyric-coverage library runs:
// the detector first, the provider behind it.
func detectorLaneFront(d detector.Detector, provider *Lane) []*Lane {
	lane := NewDetectorLane(d, circuit.New(time.Minute, time.Hour))
	return []*Lane{lane, provider}
}

// TestDetectorFirst_VocalTrackStillReachesProvider is the core no-regression
// case: with the detector running FIRST, a track the provider can serve still
// reaches the provider and the provider's lyrics win.
func TestDetectorFirst_VocalTrackStillReachesProvider(t *testing.T) {
	d := vocalTrackDetector()
	p := &stubProvider{name: "musixmatch", song: syncedSong()}
	providerLane, _ := newTestLane(p)
	lanes := detectorLaneFront(d, providerLane)

	o, err := New(ModeOrdered, lanes...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	song, err := o.FindLyrics(context.Background(), models.Track{TrackName: "x"}, "/music/x.flac")
	if err != nil {
		t.Fatalf("FindLyrics: %v", err)
	}

	if d.calls != 1 {
		t.Fatalf("detector calls = %d; want 1 (the detector still runs first)", d.calls)
	}
	if p.calls != 1 {
		t.Fatalf("provider calls = %d; want 1 -- a gate-negative verdict must NOT preempt the provider", p.calls)
	}
	if song.WinningLane != "musixmatch" {
		t.Fatalf("winning lane = %q; want musixmatch", song.WinningLane)
	}
	if song.Track.Instrumental == 1 {
		t.Fatal("a vocal track was settled as instrumental; the detector preempted a servable provider")
	}
}

// TestDetectorFirst_GateNegativeContributesNoResult asserts the detector adds
// nothing to the attribution's outcome for a vocal track: it is attempted, but
// the provider is the hit. A detector that contributed a retained best-available
// result here could win a dispatch where the provider merely returns something
// weaker, which is the regression path.
func TestDetectorFirst_GateNegativeContributesNoResult(t *testing.T) {
	d := vocalTrackDetector()
	p := &stubProvider{name: "musixmatch", song: syncedSong()}
	providerLane, _ := newTestLane(p)
	lanes := detectorLaneFront(d, providerLane)

	o, err := New(ModeOrdered, lanes...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	song, err := o.FindLyrics(context.Background(), models.Track{TrackName: "x"}, "/music/x.flac")
	if err != nil {
		t.Fatalf("FindLyrics: %v", err)
	}

	for _, a := range song.LaneAttempts {
		if a.Lane == detectorLaneName && a.Hit {
			t.Fatalf("detector recorded as the hit on a vocal track: %+v", song.LaneAttempts)
		}
	}
	if a := attemptFor(t, song, "musixmatch"); !a.Hit {
		t.Fatalf("provider attempt = %+v; want Hit true", a)
	}
}

// TestDetectorFirst_ProviderMissStillFallsBackToProviderOutcome verifies the
// reordering did not change what happens when the provider genuinely has
// nothing: a gate-negative detector must not manufacture an instrumental
// marker. Before the lane existed, this track was a plain miss; it must stay one.
func TestDetectorFirst_ProviderMissStillFallsBackToProviderOutcome(t *testing.T) {
	d := vocalTrackDetector()
	p := &stubProvider{name: "musixmatch", err: ErrLaneBenignMiss}
	providerLane, _ := newTestLane(p)
	lanes := detectorLaneFront(d, providerLane)

	o, err := New(ModeOrdered, lanes...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	song, err := o.FindLyrics(context.Background(), models.Track{TrackName: "x"}, "/music/x.flac")
	if err == nil && song.Track.Instrumental == 1 {
		t.Fatal("a gate-negative detector produced an instrumental settle on a provider miss")
	}
	if d.calls != 1 || p.calls != 1 {
		t.Fatalf("detector calls = %d, provider calls = %d; want 1 and 1", d.calls, p.calls)
	}
}

// TestDetectorFirst_GatePositiveStillPreempts is the counterweight: the tests
// above must not pass by accident on an orchestrator where the detector lane is
// inert. A gate-positive verdict MUST settle the track without calling the
// provider -- that is #501's whole point, and it is what makes the negative
// assertions above meaningful.
func TestDetectorFirst_GatePositiveStillPreempts(t *testing.T) {
	d := &stubDetector{res: detector.Result{
		Instrumental: true, Confidence: 0.9, VocalConfidence: 0.01,
		SpeechConfidence: 0.02, Version: "1.5.0",
	}}
	p := &stubProvider{name: "musixmatch", song: syncedSong()}
	providerLane, _ := newTestLane(p)
	lanes := detectorLaneFront(d, providerLane)

	o, err := New(ModeOrdered, lanes...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	song, err := o.FindLyrics(context.Background(), models.Track{TrackName: "x"}, "/music/x.flac")
	if err != nil {
		t.Fatalf("FindLyrics: %v", err)
	}
	if p.calls != 0 {
		t.Fatalf("provider calls = %d; want 0 -- a confident instrumental settles with zero provider requests", p.calls)
	}
	if song.WinningLane != detectorLaneName || song.Track.Instrumental != 1 {
		t.Fatalf("song = %+v; want a detector-settled instrumental", song)
	}
}
