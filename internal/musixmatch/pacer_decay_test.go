package musixmatch

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestAdaptiveLevelStarvesUnderRealisticMissRate reproduces the measured
// production event mix (2026-07-18, canticle 1.20.1) and asserts the pacer eases
// back toward its floor once the provider has stopped throttling.
//
// The measured window after the last ratcheting throttle contained 52 settled
// work items: 42 benign misses, 8 detector-lane instrumental settles, and only 2
// genuine Musixmatch catalog hits. Neither a benign miss nor a detector settle
// reaches this pacer's OnSuccess -- the orchestrator deliberately withholds
// notifySuccess for a benign miss (internal/orchestrator/lane.go:67 fires only
// on the success path), and (before #550) the detector lane was constructed
// with no pacer at all (internal/orchestrator/detector_lane.go). So only the 2
// catalog hits call OnSuccess, which is short of adaptiveSuccessThreshold (5),
// and the streak-based path alone never steps the level down.
//
// This test DISCRIMINATES: against the pre-#492 one-way ratchet it fails with
// level 3 (the observed 8x multiplier) held forever, and it can only pass once
// a step-down path exists that does not depend on a 5-long run of consecutive
// catalog hits. It exercises the time-decay path specifically: it injects a
// fake clock (the now/sleep seam) and drives recovery through c.pace(), the
// sole checkpoint decayLocked runs from -- not through OnSuccess, which stays
// below threshold throughout, proving decay (not the streak counter) is what
// recovers the level.
func TestAdaptiveLevelStarvesUnderRealisticMissRate(t *testing.T) {
	const (
		throttles        = 3  // measured: 3 ratcheting 401s -> level 3
		catalogHits      = 2  // measured: kind=synced saves
		benignMisses     = 42 // measured: "benign miss deferred"
		detectorSettles  = 8  // measured: kind=instrumental saves
		wantAfterRecover = 0  // provider stopped throttling; expect a return to the floor
	)

	base := time.Unix(9000, 0)
	var mu sync.Mutex
	fakeNow := base
	nowFn := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return fakeNow
	}
	sleepFn := func(_ context.Context, d time.Duration) bool {
		mu.Lock()
		fakeNow = fakeNow.Add(d)
		mu.Unlock()
		return true
	}

	c := NewClient("token")
	c.WithMinInterval(15 * time.Second)
	c.now = nowFn
	c.sleep = sleepFn

	for range throttles {
		c.OnThrottle()
	}
	if got := readAdaptiveLevel(c); got != adaptiveMaxLevel {
		t.Fatalf("after %d throttles: level = %d; want %d (precondition)", throttles, got, adaptiveMaxLevel)
	}

	// Replay the post-throttle window. Benign misses and detector settles are
	// deliberately NOT delivered to the pacer's OnSuccess -- that is what
	// production did before #550, and modeling them as OnSuccess here would fake
	// a recovery the pre-fix system never got. They are counted only to
	// document the true settle volume.
	settled := benignMisses + detectorSettles
	for range catalogHits {
		c.OnSuccess()
	}
	if got := readConsecutiveSuccesses(c); got != catalogHits {
		t.Fatalf("success counter = %d; want %d (precondition: must stay below the %d threshold, so any recovery below comes from decay, not the streak path)",
			got, catalogHits, adaptiveSuccessThreshold)
	}

	// Advance the clock past the full multi-level decay window and let pace()
	// evaluate it lazily -- exactly as the next queued item's FindLyrics call
	// would in production. No goroutine, no ticker: pace() is the sole
	// checkpoint (decayLocked).
	mu.Lock()
	fakeNow = fakeNow.Add(adaptiveMaxLevel * adaptiveDecayInterval)
	mu.Unlock()
	if err := c.pace(context.Background()); err != nil {
		t.Fatalf("pace: %v", err)
	}

	if got := readAdaptiveLevel(c); got != wantAfterRecover {
		t.Errorf("after a throttle-free window of %d settled items (%d catalog hits, %d benign misses, %d detector settles) spanning %d decay intervals: level = %d, multiplier = %dx; want level %d",
			settled+catalogHits, catalogHits, benignMisses, detectorSettles, adaptiveMaxLevel, got, 1<<got, wantAfterRecover)
	}
}

// TestDecayLockedStepsOneLevelPerInterval pins the STEPPING behavior of
// decayLocked, not just its eventual endpoint: it must ease the level down by
// exactly one per full adaptiveDecayInterval elapsed, never jump straight to
// the floor. The comment above adaptiveDecayInterval justifies its 20-minute
// value specifically because a full unwind from adaptiveMaxLevel takes 3x
// that -- "long enough that a short lull between genuine 401s does not
// immediately erase the ratchet and re-provoke the provider." A slam-to-zero
// implementation (reset to 0 the instant any decay window elapses, rather
// than stepping down one level at a time) defeats exactly that property:
// after a single 20-minute lull the level would go straight from 3 to 0
// instead of 3 to 2. TestAdaptiveLevelStarvesUnderRealisticMissRate cannot
// catch this because it advances the clock by the whole
// adaptiveMaxLevel*adaptiveDecayInterval window in one jump, which cannot
// distinguish gradual stepping from an instant reset.
//
// This test drives decay through c.pace() (the same now/sleep seam
// TestAdaptiveLevelStarvesUnderRealisticMissRate uses -- decayLocked has no
// other caller), advancing the fake clock by ONE adaptiveDecayInterval per
// pace() call and asserting the level after each step.
func TestDecayLockedStepsOneLevelPerInterval(t *testing.T) {
	base := time.Unix(9000, 0)
	var mu sync.Mutex
	fakeNow := base
	nowFn := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return fakeNow
	}
	sleepFn := func(_ context.Context, d time.Duration) bool {
		mu.Lock()
		fakeNow = fakeNow.Add(d)
		mu.Unlock()
		return true
	}

	c := NewClient("token")
	c.WithMinInterval(15 * time.Second)
	c.now = nowFn
	c.sleep = sleepFn

	for range adaptiveMaxLevel {
		c.OnThrottle()
	}
	if got := readAdaptiveLevel(c); got != adaptiveMaxLevel {
		t.Fatalf("after %d throttles: level = %d; want %d (precondition)", adaptiveMaxLevel, got, adaptiveMaxLevel)
	}

	// Advancing less than a full interval must not step the level at all.
	mu.Lock()
	fakeNow = fakeNow.Add(adaptiveDecayInterval - time.Second)
	mu.Unlock()
	if err := c.pace(context.Background()); err != nil {
		t.Fatalf("pace: %v", err)
	}
	if got := readAdaptiveLevel(c); got != adaptiveMaxLevel {
		t.Fatalf("after less than one full decay interval: level = %d; want unchanged %d", got, adaptiveMaxLevel)
	}

	// One more second completes the first full interval: level steps down by
	// exactly one, not to zero.
	wantLevels := []int{adaptiveMaxLevel - 1, adaptiveMaxLevel - 2, 0, 0}
	for i, want := range wantLevels {
		mu.Lock()
		fakeNow = fakeNow.Add(time.Second)
		mu.Unlock()
		if err := c.pace(context.Background()); err != nil {
			t.Fatalf("pace (step %d): %v", i, err)
		}
		if got := readAdaptiveLevel(c); got != want {
			t.Fatalf("step %d (after %d full decay interval(s) plus this one second): level = %d; want %d",
				i, i+1, got, want)
		}
		if i < len(wantLevels)-1 {
			mu.Lock()
			fakeNow = fakeNow.Add(adaptiveDecayInterval - time.Second)
			mu.Unlock()
		}
	}

	// A fourth interval beyond the unwind must not underflow below the floor.
	mu.Lock()
	fakeNow = fakeNow.Add(adaptiveDecayInterval)
	mu.Unlock()
	if err := c.pace(context.Background()); err != nil {
		t.Fatalf("pace (floor check): %v", err)
	}
	if got := readAdaptiveLevel(c); got != 0 {
		t.Fatalf("after a decay interval beyond full unwind: level = %d; want 0 (floor, no underflow)", got)
	}
}
