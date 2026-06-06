package commands

import "fmt"

// Build-time version metadata. These defaults are overridden at release time by
// GoReleaser via -ldflags "-X .../internal/commands.version=..." (see the
// builds section of .goreleaser.yml). A plain `go build`, `go run`, or the
// nightly/CVE-scan image leaves the defaults, so a non-release binary reports a
// clear "dev" marker instead of masquerading as a tagged version.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// VersionString renders the single line printed for --version.
func VersionString() string {
	return fmt.Sprintf("mxlrcgo-svc %s (commit %s, built %s)", version, commit, date)
}

// Version implements go-arg's Versioned interface for the subcommand-aware
// parser, so `mxlrcgo-svc <cmd> --version` is recognized.
func (Args) Version() string { return VersionString() }

// Version implements go-arg's Versioned interface for the legacy (no
// subcommand) parser, so top-level `mxlrcgo-svc --version` works too. Both
// parse targets need the method because Run selects the target based on whether
// the first argument names a subcommand.
func (LegacyArgs) Version() string { return VersionString() }
