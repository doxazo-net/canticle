package normalize

import "strings"

// ResolveArtist picks the primary artist to use for lyric matching. The
// album-artist tag is preferred because it is, for most releases, the cleanest
// single primary-artist string and is free of the multi-value concatenation
// that the track-artist tag can carry (e.g. FLAC repeated ARTIST comments or
// ID3v2.4 null-joined values). Generic compilation placeholders are NOT the
// track's artist, so they fall back to the track artist instead.
//
// The chosen value is returned verbatim (not trimmed/normalized) so callers
// retain the original tag text; key normalization happens separately via
// NormalizeKey.
func ResolveArtist(albumArtist, artist string) string {
	trimmed := strings.TrimSpace(albumArtist)
	if trimmed != "" && !isGenericAlbumArtist(trimmed) {
		return albumArtist
	}
	return artist
}

// isGenericAlbumArtist reports whether s is a compilation/various-artists
// placeholder that should not be used as the matching artist.
func isGenericAlbumArtist(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "various artists", "various", "va":
		return true
	default:
		return false
	}
}
