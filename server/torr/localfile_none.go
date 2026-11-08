//go:build !full

package torr

import (
	"net/http"

	"github.com/anacrolix/torrent"
)

// streamDownloaded: --downloaddir is in the full profile only (localfile.go).
func (t *Torrent) streamDownloaded(int, *torrent.File, *http.Request, http.ResponseWriter, int) (bool, error) {
	return false, nil
}
