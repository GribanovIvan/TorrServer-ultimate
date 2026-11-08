//go:build full

package settings

// FullSets are the settings of features only the full profile has.
type FullSets struct {
	PerUserData bool // per-user data separation (freetushkan), see peruser.go

	// External subtitles (/subs, sirin-media/TorrServer-NG), see subtitles_full.go
	MaxSubtitleSizeMB          int // in MB, bigger subtitle files are ignored; 0 - default, see MaxSubtitleSize
	ExternalSubtitleTimeoutSec int // in seconds, time limit for loading the subtitle tracks of a video into HLS; 0 - default
	ExternalSubtitleAttempts   int // how many times loading may be tried for tracks that failed for a temporary reason; 0 - default
}

// normalizeFull brings the full-only settings into their ranges.
func (v *BTSets) normalizeFull() {
	v.normalizeSubtitles()
}
