package commands

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionStringUsesBuildDefaults(t *testing.T) {
	got := VersionString()
	for _, want := range []string{"canticle", version, commit, date} {
		if !strings.Contains(got, want) {
			t.Errorf("VersionString() = %q, missing %q", got, want)
		}
	}
}

func TestRunVersionFlag(t *testing.T) {
	// --version must work both at the top level (legacy parser) and after a
	// subcommand (subcommand-aware parser); both parse targets implement
	// go-arg's Versioned interface.
	for _, args := range [][]string{{"--version"}, {"serve", "--version"}} {
		var out bytes.Buffer
		code := Run(context.Background(), args, &out, Deps{})
		if code != 0 {
			t.Errorf("Run(%v) exit = %d, want 0", args, code)
		}
		if got := strings.TrimSpace(out.String()); got != VersionString() {
			t.Errorf("Run(%v) output = %q, want %q", args, got, VersionString())
		}
	}
}
