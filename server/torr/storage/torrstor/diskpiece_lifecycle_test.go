package torrstor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"server/settings"
)

func diskPieceForTest(t *testing.T, dir string, size int64) (*Cache, *DiskPiece) {
	t.Helper()
	c := NewCache(size, NewStorage(size))
	c.pieceLength = size
	p := &Piece{Id: 0, cache: c}
	d := &DiskPiece{piece: p, name: filepath.Join(dir, "0")}
	p.dPiece = d
	c.pieces[0] = p
	t.Cleanup(func() { c.Close() })
	return c, d
}

func TestDiskPieceRejectsIOAfterCacheClose(t *testing.T) {
	initTestSettings(t)
	c, d := diskPieceForTest(t, t.TempDir(), 4096)
	if _, err := d.WriteAt([]byte("original"), 0); err != nil {
		t.Fatal(err)
	}
	c.Close()
	if _, err := d.WriteAt([]byte("late"), 0); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("write after close = %v, want os.ErrNotExist", err)
	}
	if _, err := d.ReadAt(make([]byte, 1), 0); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read after close = %v, want os.ErrNotExist", err)
	}
	// An active handle must be closed even when cache files are retained.
	if err := os.Rename(d.name, d.name+".retained"); err != nil {
		t.Fatal(err)
	}
}

func TestDiskPieceEvictionClosesFileAndAllowsFreshData(t *testing.T) {
	initTestSettings(t)
	_, d := diskPieceForTest(t, t.TempDir(), 4096)
	if _, err := d.WriteAt([]byte("original"), 0); err != nil {
		t.Fatal(err)
	}
	d.Release()
	if _, err := os.Stat(d.name); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("evicted file still exists: %v", err)
	}
	if _, err := d.WriteAt([]byte("new-data"), 0); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 4)
	if _, err := d.ReadAt(b, 0); err != nil {
		t.Fatal(err)
	}
	if string(b) != "new-" {
		t.Fatalf("read %q after eviction", b)
	}
}

func TestDiskPieceConcurrentWritesPreserveBytes(t *testing.T) {
	initTestSettings(t)
	_, d := diskPieceForTest(t, t.TempDir(), 1<<20)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := bytes.Repeat([]byte{byte(i + 1)}, 16384)
			if n, err := d.WriteAt(b, int64(i*len(b))); err != nil || n != len(b) {
				t.Errorf("write: %d, %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	b := make([]byte, 8*16384-1)
	if _, err := d.ReadAt(b, 0); err != nil {
		t.Fatal(err)
	}
	for i, v := range b {
		if v != byte(i/16384+1) {
			t.Fatalf("wrong byte at %d", i)
		}
	}
}

func TestDiskPieceReadOnlyHandleCanUpgradeForWrite(t *testing.T) {
	initTestSettings(t)
	_, d := diskPieceForTest(t, t.TempDir(), 4096)
	if err := os.WriteFile(d.name, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	d.piece.Size = 8
	if _, err := d.ReadAt(make([]byte, 1), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := d.WriteAt([]byte("new"), 0); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 3)
	if _, err := d.ReadAt(b, 0); err != nil {
		t.Fatal(err)
	}
	if string(b) != "new" {
		t.Fatalf("updated bytes = %q", b)
	}
}

func TestDiskPieceConcurrentIOAndCacheClose(t *testing.T) {
	initTestSettings(t)
	c, d := diskPieceForTest(t, t.TempDir(), 1<<20)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := d.WriteAt([]byte{byte(i)}, int64(i))
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Errorf("write: %v", err)
				}
			}
		}(i)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if err := os.Rename(d.name, d.name+".closed"); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestCacheCloseRemovesDiskFilesWhenRequested(t *testing.T) {
	initTestSettings(t)
	before := *settings.BTsets
	t.Cleanup(func() { *settings.BTsets = before })
	settings.BTsets.UseDisk = true
	settings.BTsets.RemoveCacheOnDrop = true
	settings.BTsets.TorrentsSavePath = t.TempDir()
	dir := filepath.Join(settings.BTsets.TorrentsSavePath, "0000000000000000000000000000000000000000")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	c, d := diskPieceForTest(t, dir, 4096)
	if _, err := d.WriteAt([]byte("original"), 0); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory remains: %v", err)
	}
}

// This host-specific gate reproduces the measured D: bottleneck. It is opt-in
// so CI machines do not acquire an absolute storage-throughput requirement.
func TestDiskPieceSmallWritesHostBudget(t *testing.T) {
	root := os.Getenv("TORRSERVE_IO_TEST_ROOT")
	if root == "" {
		t.Skip("host storage budget is opt-in")
	}
	initTestSettings(t)
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "codex-disk-io-")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(dir) != root {
		t.Fatal("temporary directory escaped cache root")
	}
	c, d := diskPieceForTest(t, dir, 16<<20)
	defer func() { c.Close(); os.Remove(d.name); os.Remove(dir) }()
	b := bytes.Repeat([]byte{0x5a}, 16384)
	start := time.Now()
	for off := int64(0); off < 16<<20; off += int64(len(b)) {
		if n, err := d.WriteAt(b, off); n != len(b) || err != nil {
			t.Fatalf("write: %d, %v", n, err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("16MiB /16KiB writes: %s, %.2f MiB/s", elapsed, 16/elapsed.Seconds())
	if elapsed > 4*time.Second {
		t.Fatalf("small writes exceed host budget: %s >4s", elapsed)
	}
}
