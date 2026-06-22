package version

import (
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	got := VersionString()
	if !strings.HasPrefix(got, "canticle ") {
		t.Errorf("VersionString() = %q, want prefix \"canticle \"", got)
	}
	for _, sub := range []string{Version, Commit, Date} {
		if !strings.Contains(got, sub) {
			t.Errorf("VersionString() = %q, does not contain %q", got, sub)
		}
	}
}
