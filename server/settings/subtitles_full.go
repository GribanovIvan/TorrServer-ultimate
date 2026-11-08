//go:build full

package settings

import "time"

// Limits for external subtitles, from sirin-media/TorrServer-NG (ba499cfe).

const (
	// DefaultMaxSubtitleSizeMB is the default size limit of an external subtitle file.
	// Keep it in line with subs.MaxSubSize.
	DefaultMaxSubtitleSizeMB = 16
	// MaxSubtitleSizeLimitMB is the upper bound of the MaxSubtitleSizeMB setting:
	// a subtitle file is read into memory as a whole.
	MaxSubtitleSizeLimitMB = 32
)

// Defaults and upper bounds of the settings that control loading of the external subtitle
// tracks into HLS. The timeout is bounded by the 60 s that /subs itself allows for reading
// one file, and a longer one only delays the start of the playback.
const (
	DefaultExternalSubtitleTimeoutSec = 15
	MaxExternalSubtitleTimeoutSec     = 60
	DefaultExternalSubtitleAttempts   = 3
	MaxExternalSubtitleAttempts       = 10
)

// normalizeSubtitles brings the external subtitle settings into the allowed ranges:
// 0 or less means the default, anything above the upper bound is cut.
func (v *BTSets) normalizeSubtitles() {
	if v.MaxSubtitleSizeMB <= 0 {
		v.MaxSubtitleSizeMB = DefaultMaxSubtitleSizeMB
	}
	if v.MaxSubtitleSizeMB > MaxSubtitleSizeLimitMB {
		v.MaxSubtitleSizeMB = MaxSubtitleSizeLimitMB
	}
	if v.ExternalSubtitleTimeoutSec <= 0 {
		v.ExternalSubtitleTimeoutSec = DefaultExternalSubtitleTimeoutSec
	}
	if v.ExternalSubtitleTimeoutSec > MaxExternalSubtitleTimeoutSec {
		v.ExternalSubtitleTimeoutSec = MaxExternalSubtitleTimeoutSec
	}
	if v.ExternalSubtitleAttempts <= 0 {
		v.ExternalSubtitleAttempts = DefaultExternalSubtitleAttempts
	}
	if v.ExternalSubtitleAttempts > MaxExternalSubtitleAttempts {
		v.ExternalSubtitleAttempts = MaxExternalSubtitleAttempts
	}
}

// MaxSubtitleSize returns the size limit of an external subtitle file, in bytes.
func MaxSubtitleSize() int64 {
	mb := DefaultMaxSubtitleSizeMB
	if BTsets != nil && BTsets.MaxSubtitleSizeMB > 0 {
		mb = BTsets.MaxSubtitleSizeMB
	}
	if mb > MaxSubtitleSizeLimitMB {
		mb = MaxSubtitleSizeLimitMB
	}
	return int64(mb) << 20
}

// ExternalSubtitleTimeout returns the time limit for loading the external subtitle
// tracks of a video into HLS.
func ExternalSubtitleTimeout() time.Duration {
	sec := DefaultExternalSubtitleTimeoutSec
	if BTsets != nil && BTsets.ExternalSubtitleTimeoutSec > 0 {
		sec = BTsets.ExternalSubtitleTimeoutSec
	}
	if sec > MaxExternalSubtitleTimeoutSec {
		sec = MaxExternalSubtitleTimeoutSec
	}
	return time.Duration(sec) * time.Second
}

// ExternalSubtitleAttempts returns how many times loading the external subtitle tracks
// of a video may be tried, when some of them fail for a temporary reason. 1 means no retry.
func ExternalSubtitleAttempts() int {
	n := DefaultExternalSubtitleAttempts
	if BTsets != nil && BTsets.ExternalSubtitleAttempts > 0 {
		n = BTsets.ExternalSubtitleAttempts
	}
	if n > MaxExternalSubtitleAttempts {
		n = MaxExternalSubtitleAttempts
	}
	return n
}
