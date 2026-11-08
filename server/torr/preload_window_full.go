//go:build full

package torr

// Preload window from Tommixoft/TorrServer (full profile only): sized off the
// preload target and shrunk in whole-piece steps. The default profile keeps a
// fixed four-piece window (preload_window.go).

// preloadWindow returns the in-flight readahead for a preload reader.
//
// A fixed pieceLength*4 pins the pipeline to four pieces regardless of link
// speed or configured cache, so a bigger cache only makes the wait longer.
// Sizing the window off the preload target lets it scale with CacheSize and
// PreloadCache, while pieceLength*4 remains the floor for small caches. The
// result is clamped to span (the bytes this reader will actually read).
func preloadWindow(size, span, pieceLength int64) int64 {
	ra := size / 4
	if floor := pieceLength * 4; ra < floor {
		ra = floor
	}
	if ra > span {
		ra = span
	}
	return ra
}

// shrinkPreloadWindow shrinks the window to what is left, in whole-piece steps
// only: SetNext and SetReadahead take the client lock and the preload loop runs
// per 32 KB. It returns the new readahead and zone, and whether to apply them.
func shrinkPreloadWindow(readahead, remain, pieceLength int64) (int64, int64, bool) {
	if remain >= 0 && readahead-remain >= pieceLength {
		return remain, remain, true
	}
	return readahead, 0, false
}
