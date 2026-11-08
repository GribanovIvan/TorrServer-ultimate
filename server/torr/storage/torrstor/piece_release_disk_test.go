//go:build !slim

package torrstor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anacrolix/torrent/metainfo"

	"server/settings"
)

func TestPieceReleaseDiskRemovesFile(t *testing.T) {
	testBTSets(t)
	dir := t.TempDir()
	settings.BTsets.UseDisk = true
	settings.BTsets.TorrentsSavePath = dir
	updates := countCompletionUpdates(t)
	c := &Cache{
		pieces:       map[int]*Piece{},
		activePieces: map[int]struct{}{},
		hash:         metainfo.Hash{7},
		pieceLength:  8, pieceCount: 1, totalLength: 8, capacity: 1 << 20,
	}
	if err := os.MkdirAll(filepath.Join(dir, c.hash.HexString()), 0o777); err != nil {
		t.Fatal(err)
	}
	p := NewPiece(0, c)
	c.pieces[0] = p
	if _, err := p.WriteAt([]byte("abcdefgh"), 0); err != nil {
		t.Fatal(err)
	}
	if path := p.dPiece.name; path == "" {
		t.Fatal("disk piece has no file name")
	} else if _, err := os.Stat(path); err != nil {
		t.Fatalf("piece file missing before release: %v", err)
	} else {
		p.Release()
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("piece file kept after release: %v", err)
		}
	}
	if *updates != 1 {
		t.Fatalf("completion updates = %d", *updates)
	}

	var nilPiece *Piece
	if got := nilPiece.releaseBuffers(); got != "" {
		t.Fatalf("nil piece path %q", got)
	}
	if got := (&Piece{}).releaseBuffers(); got != "" {
		t.Fatalf("bare disk piece path %q", got)
	}
}

// Without a torrent pointer there is no client lock to take: evictPartial falls
// back to a plain Release.
