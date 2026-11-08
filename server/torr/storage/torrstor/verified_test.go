//go:build !slim

package torrstor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anacrolix/torrent/metainfo"

	"server/settings"
)

func verifiedCache(t *testing.T, removeOnDrop bool, pieces int) (*Cache, string) {
	t.Helper()
	testBTSets(t)
	settings.BTsets.UseDisk = true
	settings.BTsets.RemoveCacheOnDrop = removeOnDrop
	settings.BTsets.TorrentsSavePath = t.TempDir()
	c := newVerifiedCache(metainfo.Hash{9}, pieces)
	return c, filepath.Join(settings.BTsets.TorrentsSavePath, metainfo.Hash{9}.HexString())
}

func newVerifiedCache(hash metainfo.Hash, pieces int) *Cache {
	c := NewCache(1<<30, NewStorage(1<<30))
	c.pieceLength = 16
	c.pieceCount = pieces
	c.totalLength = int64(pieces) * 16
	c.hash = hash
	dir := filepath.Join(settings.BTsets.TorrentsSavePath, hash.HexString())
	_ = os.MkdirAll(dir, 0o777)
	for i := 0; i < pieces; i++ {
		c.pieces[i] = NewPiece(i, c)
	}
	c.loadVerified()
	return c
}

func writeFullPiece(t *testing.T, c *Cache, id int) {
	t.Helper()
	if _, err := c.pieces[id].WriteAt(make([]byte, 16), 0); err != nil {
		t.Fatal(err)
	}
}

func TestVerifiedPieceTrustedAfterReopen(t *testing.T) {
	c, dir := verifiedCache(t, false, 3)
	writeFullPiece(t, c, 0)
	writeFullPiece(t, c, 1)
	if err := c.pieces[0].MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, verifiedFile)); err != nil {
		t.Fatalf("sidecar missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "0")); err != nil {
		t.Fatalf("piece file removed on close without RemoveCacheOnDrop: %v", err)
	}

	re := newVerifiedCache(metainfo.Hash{9}, 3)
	t.Cleanup(func() { re.Close() })
	if got := re.pieces[0].Completion(); !got.Complete || !got.Ok {
		t.Fatalf("verified piece: %+v, want complete and ok", got)
	}
	if got := re.pieces[1].Completion(); got.Complete || got.Ok {
		t.Fatalf("unverified full piece: %+v, want not complete, Ok=false for hash check", got)
	}
	if got := re.pieces[2].Completion(); got.Complete || !got.Ok {
		t.Fatalf("absent piece: %+v, want not complete, Ok=true", got)
	}
	// Hash check passed: trusted from now on.
	if err := re.pieces[1].MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if got := re.pieces[1].Completion(); !got.Complete || !got.Ok {
		t.Fatalf("after MarkComplete: %+v", got)
	}
}

func TestHashFailClearsVerifiedAndFile(t *testing.T) {
	c, dir := verifiedCache(t, false, 1)
	t.Cleanup(func() { c.Close() })
	writeFullPiece(t, c, 0)
	if err := c.pieces[0].MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if err := c.pieces[0].MarkNotComplete(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "0")); !os.IsNotExist(err) {
		t.Fatalf("file left after hash failure: %v", err)
	}
	if c.verified[0] {
		t.Fatal("bitmap still marks a released piece")
	}
}

func TestCacheCloseRemovesVerifiedWithFiles(t *testing.T) {
	c, dir := verifiedCache(t, true, 1)
	writeFullPiece(t, c, 0)
	if err := c.pieces[0].MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("cache dir left with RemoveCacheOnDrop: %v", err)
	}
}

func TestVerifiedBitmapRoundTrip(t *testing.T) {
	in := []bool{true, false, true, true, false, false, false, false, true}
	if got := decodeVerified(encodeVerified(in), len(in)); len(got) != len(in) {
		t.Fatal(got)
	} else {
		for i := range in {
			if got[i] != in[i] {
				t.Fatalf("bit %d = %v", i, got[i])
			}
		}
	}
	if got := decodeVerified("%%%bad", 4); got[0] || got[3] {
		t.Fatalf("bad input = %v", got)
	}
}
