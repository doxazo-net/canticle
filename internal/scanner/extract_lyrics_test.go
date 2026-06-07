package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractEmbeddedLyrics(t *testing.T) {
	dir := t.TempDir()

	// Success: writes the sidecar.
	if err := extractEmbeddedLyrics(dir, "song", "hello\nworld"); err != nil {
		t.Fatalf("extract: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "song.txt"))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if string(got) != "hello\nworld" {
		t.Fatalf("sidecar = %q; want %q", string(got), "hello\nworld")
	}

	// Already exists: no-op success, must NOT overwrite the existing content.
	if err := extractEmbeddedLyrics(dir, "song", "DIFFERENT CONTENT"); err != nil {
		t.Fatalf("extract (exists): %v", err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "song.txt"))
	if string(got) != "hello\nworld" {
		t.Fatalf("sidecar overwritten = %q; want unchanged", string(got))
	}

	// Create error: a non-existent parent directory fails the temp-file create.
	if err := extractEmbeddedLyrics(filepath.Join(dir, "does-not-exist"), "x", "y"); err == nil {
		t.Fatal("extract into missing dir = nil; want an error")
	}
}
