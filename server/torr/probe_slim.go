//go:build slim

package torr

// The slim profile does not run ffprobe: bitrate and duration stay empty.
func probeMedia(*Torrent, string, int) {}
