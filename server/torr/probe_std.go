//go:build !slim

package torr

import (
	"strconv"

	"server/ffprobe"
	"server/settings"
)

// probeMedia reads bitrate and duration of the preloaded file with ffprobe, when
// it is installed, for the torrent status.
func probeMedia(t *Torrent, hash string, index int) {
	if !ffprobe.Exists() {
		return
	}
	link := settings.LoopbackBaseURL() + "/play/" + hash + "/" + strconv.Itoa(index)
	if data, err := ffprobe.ProbeUrl(link); err == nil {
		t.BitRate = data.Format.BitRate
		t.DurationSeconds = data.Format.DurationSeconds
	}
}
