package orchestrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/sydlexius/canticle/internal/circuit"
	"github.com/sydlexius/canticle/internal/detector"
	"github.com/sydlexius/canticle/internal/models"
	"github.com/sydlexius/canticle/internal/providers"
)

// detectorLaneName is the lane name a detector-backed lane reports.
const detectorLaneName = "detector"

// NewDetectorLane builds an orchestrator lane over the audio detector and its
// dedicated breaker. The resolve func runs the 3-gate over the work item's audio
// path: a gate-positive verdict returns a terminal-suitable instrumental Song
// carrying detector telemetry; a gate-negative verdict (or an empty path)
// returns ErrLaneBenignMiss so the providers run; a detector call failure
// returns ErrLaneOutage so the breaker trips.
//
// pacer is optional (nil is valid, and every existing call site predating #550
// passed none): when non-nil it is credited on a gate-positive settle so the
// detector's throughput helps ease the SHARED provider pacer's ratchet back
// down (see the local/pacer split below). Callers should pass the SAME pacer
// instance the primary provider lane uses (Lane.Pacer() on that lane), not a
// detector-owned one -- the detector has no throttle concept of its own; it is
// only ever crediting recovery of the provider's adaptive interval.
func NewDetectorLane(d detector.Detector, breaker *circuit.Breaker, pacer providers.AdaptivePacer) *Lane {
	return &Lane{
		name:        detectorLaneName,
		breaker:     breaker,
		classifyErr: detectorClassifier,
		// local=true: the detector settles a track from local audio analysis, no
		// outbound provider request, so an item settled here must not SPEND the
		// provider-request pacing budget (#534) -- Lane.FindLyrics never calls
		// pace() for this lane regardless of pacer below.
		local: true,
		// pacer decision (#550): a local lane SHOULD still CREDIT decay. Spending
		// and crediting are separable: local=true controls only whether pace() is
		// paid (it is not, for any lane -- pace() is invoked exclusively inside the
		// musixmatch client's own FindLyrics, never by Lane), while pacer here
		// controls only whether Lane.notifySuccess() fires OnSuccess on a genuine
		// settle. A detector settle proves the work item was disposed of without
		// needing a provider round-trip at all, so crediting it costs nothing (no
		// extra request is made to earn the credit) and helps the ratchet unwind
		// faster when detector settles make up a large share of traffic, exactly
		// the measured production gap (8 of 52 settles in the starved window were
		// detector instrumentals, per #492's pacer_decay_test.go). The next local
		// lane added to this package should make the same call explicitly rather
		// than silently inheriting whatever this constructor happens to do.
		pacer: pacer,
		resolve: func(ctx context.Context, track models.Track, sourcePath string) (models.Song, error) {
			// An empty sourcePath means instrumental detection is disabled for this
			// item (e.g. no audio path on the work item); the detector must never be
			// invoked in that case, so this is checked before calling Detect.
			if sourcePath == "" {
				return models.Song{}, ErrLaneBenignMiss
			}
			res, err := d.Detect(ctx, sourcePath)
			if err != nil {
				// Join rather than %v so BOTH the sentinel and the detector's own
				// cause stay matchable with errors.Is: the classifier keys on
				// ErrLaneOutage, while callers and logs need the underlying failure.
				return models.Song{}, fmt.Errorf("detector request failed: %w", errors.Join(ErrLaneOutage, err))
			}
			if !res.Instrumental {
				return models.Song{}, ErrLaneBenignMiss
			}
			// Only these identity fields are carried, deliberately: this mirrors the
			// inline miss-branch path in the worker that this lane replaces, so the
			// settled song is byte-identical to today's. Note it therefore also
			// inherits that path's limitation of dropping the remaining
			// models.Track fields (AlbumArtist, TrackLength, HasLyrics,
			// HasSubtitles, ISRC, SpotifyID, RecordingMBID). Widening this to copy
			// the incoming track is a behavior change, tracked separately, not a
			// silent tweak to make here.
			return models.Song{
				Track: models.Track{
					ArtistName:   track.ArtistName,
					TrackName:    track.TrackName,
					AlbumName:    track.AlbumName,
					Instrumental: 1,
				},
				DetectorVersion:    res.Version,
				DetectorMusicSum:   res.Confidence,
				DetectorVocalPeak:  res.VocalConfidence,
				DetectorSpeechMean: res.SpeechConfidence,
				DetectorVocalClass: res.WinningVocalClass,
			}, nil
		},
	}
}
