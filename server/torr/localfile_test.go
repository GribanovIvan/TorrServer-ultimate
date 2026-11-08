//go:build full

package torr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDownloadPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "show"), 0o777); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "show", "e01.mkv")
	if err := os.WriteFile(file, []byte("12345"), 0o666); err != nil {
		t.Fatal(err)
	}

	if got, ok := localDownloadPath(dir, "show/e01.mkv", 5); !ok || got != file {
		t.Fatalf("exact match: %q %v", got, ok)
	}
	if _, ok := localDownloadPath("", "show/e01.mkv", 5); ok {
		t.Fatal("feature must stay off without --downloaddir")
	}
	if _, ok := localDownloadPath(dir, "show/e01.mkv", 6); ok {
		t.Fatal("size mismatch (incomplete file) must fall back to the swarm")
	}
	if _, ok := localDownloadPath(dir, "show/missing.mkv", 5); ok {
		t.Fatal("missing file")
	}
	if _, ok := localDownloadPath(dir, "show", 5); ok {
		t.Fatal("a directory is not a file to serve")
	}
	outside := filepath.Join(filepath.Dir(dir), "secret")
	if err := os.WriteFile(outside, []byte("12345"), 0o666); err == nil {
		defer func() { _ = os.Remove(outside) }()
		if _, ok := localDownloadPath(dir, "../secret", 5); ok {
			t.Fatal("path traversal from torrent metadata must be rejected")
		}
	}
}
