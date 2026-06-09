package providers

import (
	"context"
	"testing"

	"github.com/sydlexius/mxlrcgo-svc/internal/models"
)

type fakeFetcher struct{}

func (fakeFetcher) FindLyrics(context.Context, models.Track) (models.Song, error) {
	return models.Song{}, nil
}

func TestSelectDefaultsToMusixmatch(t *testing.T) {
	p, err := Select("", nil, New(Musixmatch, fakeFetcher{}))
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if p.Name() != Musixmatch {
		t.Fatalf("provider = %q; want %q", p.Name(), Musixmatch)
	}
}

func TestSelectRejectsDisabledProvider(t *testing.T) {
	_, err := Select(Musixmatch, []string{" MUSIXMATCH "}, New(Musixmatch, fakeFetcher{}))
	if err == nil {
		t.Fatal("Select returned nil error; want disabled provider error")
	}
}

func TestSelectRejectsUnsupportedProvider(t *testing.T) {
	_, err := Select("future", nil, New(Musixmatch, fakeFetcher{}))
	if err == nil {
		t.Fatal("Select returned nil error; want unsupported provider error")
	}
}

func TestKnown(t *testing.T) {
	known := Known()
	if len(known) != 2 {
		t.Fatalf("Known() = %v; want exactly the two built-in providers", known)
	}
	if known[0] != Musixmatch || known[1] != PetitLyrics {
		t.Fatalf("Known() = %v; want [%q %q]", known, Musixmatch, PetitLyrics)
	}
}

func TestIsKnown(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"musixmatch", true},
		{" PetitLyrics ", true}, // case-insensitive, trimmed
		{"bogus", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsKnown(c.name); got != c.want {
			t.Errorf("IsKnown(%q) = %v; want %v", c.name, got, c.want)
		}
	}
}
