//go:build full

package torr

import "testing"

func TestPreloadWindow(t *testing.T) {
	const mb = 1 << 20
	tests := []struct {
		name                    string
		size, span, pieceLength int64
		want                    int64
	}{
		{"quarter of target wins over floor", 128 * mb, 120 * mb, 2 * mb, 32 * mb},
		{"floor wins, then clamped to span", 32 * mb, 16 * mb, 16 * mb, 16 * mb},
		{"floor wins within span", 32 * mb, 24 * mb, 4 * mb, 16 * mb},
		{"span smaller than quarter", 128 * mb, 8 * mb, 2 * mb, 8 * mb},
		{"zero span", 128 * mb, 0, 2 * mb, 0},
		{"end chunk span equals startend", 128 * mb, 8 * mb, 1 * mb, 8 * mb},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := preloadWindow(tc.size, tc.span, tc.pieceLength); got != tc.want {
				t.Fatalf("preloadWindow(%d, %d, %d) = %d, want %d", tc.size, tc.span, tc.pieceLength, got, tc.want)
			}
		})
	}
}
