package testutil

import (
	"bytes"
	"testing"

	"github.com/dhowden/tag"
)

func TestGenerateID3v2RoundTrip(t *testing.T) {
	data := GenerateID3v2("Artøst", "Tîtle", "Albüm", "line one\nline two")
	m, err := tag.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got := m.Artist(); got != "Artøst" {
		t.Errorf("Artist() = %q; want %q", got, "Artøst")
	}
	if got := m.Title(); got != "Tîtle" {
		t.Errorf("Title() = %q; want %q", got, "Tîtle")
	}
	if got := m.Album(); got != "Albüm" {
		t.Errorf("Album() = %q; want %q", got, "Albüm")
	}
	if got := m.Lyrics(); got != "line one\nline two" {
		t.Errorf("Lyrics() = %q; want %q", got, "line one\nline two")
	}
}

func TestGenerateID3v2NoLyrics(t *testing.T) {
	data := GenerateID3v2("A", "B", "C", "")
	m, err := tag.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got := m.Lyrics(); got != "" {
		t.Errorf("Lyrics() = %q; want empty (no USLT frame)", got)
	}
}
