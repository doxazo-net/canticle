package lrcnormalize

import "testing"

func FuzzParseBody(f *testing.F) {
	f.Add("")
	f.Add("[00:15.05]Hello world")
	f.Add("[00:30.00][01:05.00][02:10.00]Chorus")
	f.Add("[02:14.00][00:45.00]out of order\n[01:00.00]middle")
	f.Add("[ar:Artist]\n[ti:Title]\n[length:03:21]\n[00:01.00]line")
	f.Add("orphan\n\n[00:12.345]ms\n[01:07]nofrac")
	f.Add("[[[[00:00.00]]]]")
	f.Add("[00:12.00] [0:00]0000") // whitespace-separated stamp (regression: trim must not re-expose it)
	f.Add("[00:00.00]   [00:01.00]  x  ")

	f.Fuzz(func(t *testing.T, body string) {
		doc := ParseBody(body)

		var prev float64 = -1
		for i, c := range doc.Cues {
			// Cues are sorted ascending by timestamp.
			if c.Time.Total < prev {
				t.Fatalf("cue %d out of order: %v < %v", i, c.Time.Total, prev)
			}
			prev = c.Time.Total

			// Timestamp is internally consistent and in range.
			if c.Time.Hundredths < 0 || c.Time.Hundredths > 99 {
				t.Fatalf("cue %d hundredths out of range: %d", i, c.Time.Hundredths)
			}
			want := float64(c.Time.Minutes*60+c.Time.Seconds) + float64(c.Time.Hundredths)/100.0
			if c.Time.Total != want {
				t.Fatalf("cue %d total %v != derived %v", i, c.Time.Total, want)
			}

			// Full expansion: no cue text still begins with a leading timestamp
			// (this is the exact defect the normalizer exists to remove).
			if tsRe.MatchString(c.Text) {
				t.Fatalf("cue %d text still carries a leading timestamp: %q", i, c.Text)
			}
		}
	})
}
