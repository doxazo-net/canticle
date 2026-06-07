// Package testutil provides helpers for generating synthetic tagged audio files
// used by load/concurrency tests and the genlib tool. It encodes a minimal
// ID3v2.4 (UTF-8) tag in pure Go so no audio-encoding dependency is needed; the
// output is parseable by github.com/dhowden/tag.
package testutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// synchsafe encodes n as a 4-byte ID3v2 synchsafe integer (7 bits per byte).
func synchsafe(n int) []byte {
	return []byte{
		byte((n >> 21) & 0x7f),
		byte((n >> 14) & 0x7f),
		byte((n >> 7) & 0x7f),
		byte(n & 0x7f),
	}
}

// frame builds one ID3v2.4 frame: 4-byte id, synchsafe size, 2 flag bytes, payload.
func frame(id string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString(id)
	b.Write(synchsafe(len(payload)))
	b.Write([]byte{0x00, 0x00})
	b.Write(payload)
	return b.Bytes()
}

// textFrame builds a UTF-8 (encoding 0x03) text information frame.
func textFrame(id, text string) []byte {
	return frame(id, append([]byte{0x03}, []byte(text)...))
}

// usltFrame builds an Unsynchronized lyrics/text (USLT) frame: UTF-8 encoding,
// "eng" language, an empty content descriptor, then the lyrics. SYLT
// (Synchronized lyrics) is intentionally NOT generated -- synced-tag support is
// out of scope.
func usltFrame(lyrics string) []byte {
	var p bytes.Buffer
	p.WriteByte(0x03) // UTF-8
	p.WriteString("eng")
	p.WriteByte(0x00) // empty content descriptor terminator
	p.WriteString(lyrics)
	return frame("USLT", p.Bytes())
}

// GenerateID3v2 builds an ID3v2.4 (UTF-8) tag with TIT2 (title), TPE1 (artist),
// and TALB (album) frames. When lyrics is non-empty a USLT frame is appended.
// The returned bytes are a complete, parseable tag suitable to write as a .mp3
// file for github.com/dhowden/tag to read. Unicode artist/title/album are
// supported via the UTF-8 text encoding byte.
func GenerateID3v2(artist, title, album, lyrics string) []byte {
	var frames bytes.Buffer
	if title != "" {
		frames.Write(textFrame("TIT2", title))
	}
	if artist != "" {
		frames.Write(textFrame("TPE1", artist))
	}
	if album != "" {
		frames.Write(textFrame("TALB", album))
	}
	if lyrics != "" {
		frames.Write(usltFrame(lyrics))
	}
	var b bytes.Buffer
	b.WriteString("ID3")
	b.Write([]byte{0x04, 0x00}) // version 2.4.0
	b.WriteByte(0x00)           // header flags
	b.Write(synchsafe(frames.Len()))
	b.Write(frames.Bytes())
	return b.Bytes()
}

// WriteAudioFile writes a synthetic tagged .mp3 (ID3v2.4) into dir. lyrics is
// optional; when empty, no USLT frame is embedded.
func WriteAudioFile(dir, filename, artist, title, album, lyrics string) error {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, GenerateID3v2(artist, title, album, lyrics), 0o644); err != nil { //nolint:gosec // test fixture file
		return fmt.Errorf("testutil: write audio file %s: %w", path, err)
	}
	return nil
}

// WriteLRCFile writes a stub .lrc sidecar into dir (used to simulate libraries
// where some tracks already have lyrics).
func WriteLRCFile(dir, filename string) error {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("[00:00.00] stub\n"), 0o644); err != nil { //nolint:gosec // test fixture file
		return fmt.Errorf("testutil: write lrc file %s: %w", path, err)
	}
	return nil
}
