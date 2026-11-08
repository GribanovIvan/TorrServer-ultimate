//go:build !full

package torr

// preloadWindow is the in-flight readahead of the preload reader: four pieces,
// or none when the head span is shorter than that.
func preloadWindow(_, span, pieceLength int64) int64 {
	readahead := pieceLength * 4
	if span < readahead {
		return 0
	}
	return readahead
}

// shrinkPreloadWindow drops the readahead once fewer bytes than it are left to
// read, and narrows the zone to the rest.
func shrinkPreloadWindow(readahead, remain, _ int64) (int64, int64, bool) {
	if readahead > 0 && remain < readahead {
		if remain < 0 {
			remain = 0
		}
		return 0, remain, true
	}
	return readahead, 0, false
}
