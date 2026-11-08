package torr

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/missinggo/v2/httptoo"
	"github.com/anacrolix/torrent"

	mt "server/mimetype"
	sets "server/settings"
)

// localDownloadPath returns the file to serve from --downloaddir instead of the
// swarm, when <dir>/<torrent file path> exists with the torrent file's exact size.
// A crafted torrent path cannot leave dir.
func localDownloadPath(dir, torrentPath string, length int64) (string, bool) {
	if dir == "" || torrentPath == "" {
		return "", false
	}
	local := filepath.Join(dir, torrentPath)
	rel, err := filepath.Rel(dir, local)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	fi, err := os.Stat(local)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() != length {
		return "", false
	}
	return local, true
}

// serveDownloaded streams a file already on disk (e.g. downloaded by qBittorrent):
// no reader on the torrent, so nothing is fetched from the swarm for it.
func (t *Torrent) serveDownloaded(resp http.ResponseWriter, req *http.Request, file *torrent.File, fileID int, local string, streamTimeout int) error {
	f, err := os.Open(local)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return err
	}

	if sets.BTsets.EnableDebug {
		fmt.Println("Stream: serving local file", local)
	}
	var timecode float64
	for _, v := range sets.ListViewed(t.Hash().HexString()) {
		if v.FileIndex == fileID {
			timecode = v.TimeCode
			break
		}
	}
	sets.SetViewed(&sets.Viewed{Hash: t.Hash().HexString(), FileIndex: fileID, TimeCode: timecode})

	resp.Header().Set("Connection", "close")
	resp.Header().Set("Server", "TorrServer (Portable SDK for UPnP devices)")
	if streamTimeout > 0 {
		resp.Header().Set("X-Stream-Timeout", fmt.Sprintf("%d", streamTimeout))
	}
	etag := hex.EncodeToString([]byte(fmt.Sprintf("%s/%s", t.Hash().HexString(), file.Path())))
	resp.Header().Set("ETag", httptoo.EncodeQuotedString(etag))
	resp.Header().Set("transferMode.dlna.org", "Streaming")
	if mime, err := mt.MimeTypeByPath(file.Path()); err == nil && mime.IsMedia() {
		resp.Header().Set("content-type", mime.String())
	}
	if req.Header.Get("getContentFeatures.dlna.org") != "" {
		resp.Header().Set("contentFeatures.dlna.org", dlna.ContentFeatures{
			SupportRange:    true,
			SupportTimeSeek: true,
		}.String())
	}
	if req.Header.Get("Range") != "" {
		resp.Header().Set("Accept-Ranges", "bytes")
	}
	ServeStreamContent(resp, req, file.Path(), fi.ModTime(), f)
	return nil
}
