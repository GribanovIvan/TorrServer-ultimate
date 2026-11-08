//go:build !slim

package torrstor

import "testing"

func TestDiskHandleTableCapsOpenHandles(t *testing.T) {
	initTestSettings(t)
	saved := diskHandles
	diskHandles = newDiskHandleTable(3)
	t.Cleanup(func() { diskHandles = saved })

	dir := t.TempDir()
	pieces := make([]*DiskPiece, 0, 8)
	for i := 0; i < 8; i++ {
		_, d := diskPieceForTest(t, dir, 4096)
		d.name = d.name + string(rune('a'+i))
		if _, err := d.WriteAt([]byte{byte(i), byte(i + 1)}, 0); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		pieces = append(pieces, d)
	}
	if n := diskHandles.order.Len(); n > 3 {
		t.Fatalf("table holds %d handles, limit is 3", n)
	}
	open := 0
	for _, d := range pieces {
		if d.file != nil {
			open++
		}
	}
	if open > 3 {
		t.Fatalf("%d handles open, limit is 3", open)
	}
	// A piece whose handle was closed reopens transparently.
	for i, d := range pieces {
		b := make([]byte, 2)
		if _, err := d.ReadAt(b, 0); err != nil {
			t.Fatalf("read %d after cap: %v", i, err)
		}
		if b[0] != byte(i) || b[1] != byte(i+1) {
			t.Fatalf("piece %d read back %v", i, b)
		}
	}
}

func TestDiskHandleVictimBusyIsLeftAlone(t *testing.T) {
	initTestSettings(t)
	saved := diskHandles
	diskHandles = newDiskHandleTable(1)
	t.Cleanup(func() { diskHandles = saved })

	dir := t.TempDir()
	_, first := diskPieceForTest(t, dir, 4096)
	first.name += "1"
	if _, err := first.WriteAt([]byte("x"), 0); err != nil {
		t.Fatal(err)
	}
	first.mu.RLock() // busy: TryLock must fail, no deadlock
	_, second := diskPieceForTest(t, dir, 4096)
	second.name += "2"
	if _, err := second.WriteAt([]byte("y"), 0); err != nil {
		first.mu.RUnlock()
		t.Fatal(err)
	}
	first.mu.RUnlock()
	if first.file == nil {
		t.Fatal("busy victim handle was closed")
	}
}
