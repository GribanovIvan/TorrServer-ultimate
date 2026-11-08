//go:build slim

package offlinetest

// The slim profile has no ffprobe to hide.
func hideFFprobe(string) func() { return func() {} }
