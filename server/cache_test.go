package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveAllFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	removeAllFiles(sub)
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Fatal("directory still exists")
	}
	removeAllFiles(filepath.Join(dir, "missing"))
}
