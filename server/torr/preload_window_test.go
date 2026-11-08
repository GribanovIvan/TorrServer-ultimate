//go:build !full

package torr

import "testing"

func TestPreloadWindowFixed(t *testing.T) {
	const mb = 1 << 20
	if got := preloadWindow(128*mb, 120*mb, 2*mb); got != 8*mb {
		t.Fatalf("window %d, want four pieces", got)
	}
	if got := preloadWindow(128*mb, 4*mb, 2*mb); got != 0 {
		t.Fatalf("span below four pieces: %d, want 0", got)
	}
	if ra, zone, ok := shrinkPreloadWindow(8*mb, 9*mb, 2*mb); ok || ra != 8*mb || zone != 0 {
		t.Fatalf("no shrink expected: %d %d %v", ra, zone, ok)
	}
	if ra, zone, ok := shrinkPreloadWindow(8*mb, 3*mb, 2*mb); !ok || ra != 0 || zone != 3*mb {
		t.Fatalf("shrink: %d %d %v", ra, zone, ok)
	}
	if ra, zone, ok := shrinkPreloadWindow(8*mb, -1, 2*mb); !ok || ra != 0 || zone != 0 {
		t.Fatalf("past the end: %d %d %v", ra, zone, ok)
	}
}
