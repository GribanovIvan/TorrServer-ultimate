package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	sets "server/settings"
	"server/subs"
)

type subTrackJS struct {
	subs.Track
	URL string `json:"url"` // relative link: prepend scheme and host
}

// subsTracks godoc
//
//	@Summary		External subtitles of a video file
//	@Description	Without "track": list external subtitle tracks found in the torrent for the given video file.
//	@Description	With "track": return that subtitle file, by default converted to WebVTT.
//	@Description	"format": "vtt" (default) or "original" (no conversion). "encoding": output charset, default utf8.
//	@Description	With "fonts": list the fonts for ASS subtitles of the video: [{source, name, type, size, url}].
//	@Description	With "font": the bytes of a font attached to the video file (Matroska).
//	@Description	With "embedded": list the subtitle tracks inside the video file (Matroska), or one track with "embedded=N"
//	@Description	(optional "format": ass or vtt; "from" and "to": a window in seconds).
//
//	@Tags			API
//
//	@Param			hash	path	string	true	"Torrent infohash"
//	@Param			id		path	string	true	"Video file index in torrent"
//	@Param			track	query	int		false	"Subtitle file index (from the list)"
//	@Param			format	query	string	false	"vtt (default) or original"
//	@Param			encoding	query	string	false	"Output encoding, default utf8 (also: windows-1251)"
//	@Param			fonts	query	string	false	"Present (no value): list the fonts of the video (font files of the torrent near it and attached to the video file)"
//	@Param			font	query	int		false	"Index of a font attached to the video file (from the list of fonts)"
//	@Param			embedded	query	string	false	"Present (no value): list the subtitle tracks inside the video file; a number: that track"
//	@Param			from	query	number	false	"With embedded=N: start of the window, seconds"
//	@Param			to	query	number	false	"With embedded=N: end of the window, seconds; without it the whole track (reads the whole file)"
//
//	@Produce		json
//	@Produce		text/vtt
//	@Success		200	"List of tracks (JSON) or WebVTT"
//	@Failure		413	"Subtitle file is over the size limit"
//	@Router			/subs/{hash}/{id} [get]
func subsTracks(c *gin.Context) {
	vc, ok := loadVideoContext(c)
	if !ok {
		return
	}
	tor := vc.tor

	// Fonts for ASS subtitles: ?fonts (list: files of the torrent and attachments of a Matroska file)
	// or ?font=N (the bytes of an attached font).
	_, fonts := c.GetQuery("fonts")
	_, font := c.GetQuery("font")
	if fonts || font {
		subsFonts(c, tor, vc.infoHash, vc.videoID, vc.video, vc.files)
		return
	}

	// Embedded subtitles of a Matroska file: ?embedded (list) or ?embedded=N (one track).
	if _, ok := c.GetQuery("embedded"); ok {
		embeddedSubs(c, tor, vc.infoHash, vc.videoID, vc.video)
		return
	}

	maxSize := sets.MaxSubtitleSize()
	tracks, tooLarge := subs.MatchLimit(vc.video, vc.files, maxSize)

	// Mode 1: list the tracks.
	trackStr := c.Query("track")
	if trackStr == "" {
		// Say why a subtitle file of this video is not in the list.
		for _, t := range tooLarge {
			log.Printf("subs: skip %q: %d bytes is over the limit of %d bytes (setting MaxSubtitleSizeMB)", t.Name, t.Size, maxSize)
		}
		base := "/subs/" + vc.infoHash + "/" + strconv.Itoa(vc.videoID)
		out := make([]subTrackJS, 0, len(tracks))
		for _, t := range tracks {
			out = append(out, subTrackJS{Track: t, URL: base + "?track=" + strconv.Itoa(t.FileIndex)})
		}
		c.JSON(http.StatusOK, out)
		return
	}

	// Mode 2: one track (WebVTT or original). Only tracks found by Match are served.
	trackID, err := strconv.Atoi(trackStr)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("\"track\" is wrong"))
		return
	}
	var track *subs.Track
	for i := range tracks {
		if tracks[i].FileIndex == trackID {
			track = &tracks[i]
			break
		}
	}
	if track == nil {
		// A file of this video that is over the size limit: say so instead of 404.
		for _, t := range tooLarge {
			if t.FileIndex == trackID {
				c.AbortWithError(http.StatusRequestEntityTooLarge, errors.New("subtitle file is over the size limit (setting MaxSubtitleSizeMB)"))
				return
			}
		}
		c.AbortWithError(http.StatusNotFound, errors.New("subtitle track not found"))
		return
	}

	// Output options (validated before reading the file, so we do not wait for the torrent in vain):
	//   format   - "vtt" (default) or "original";
	//   encoding - response encoding, utf8 by default (see subs.Encode).
	format := strings.ToLower(c.DefaultQuery("format", "vtt"))
	if format != "vtt" && format != "original" {
		c.AbortWithError(http.StatusBadRequest, errors.New("\"format\" must be \"vtt\" or \"original\""))
		return
	}
	encoding := c.DefaultQuery("encoding", "utf8")
	if _, _, err := subs.Encode("", encoding); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	started := time.Now()
	raw, err := tor.ReadFile(ctx, track.FileIndex, maxSize)
	if err != nil {
		log.Printf("subs: read file %d (%s) failed after %s: %v", track.FileIndex, track.Name, time.Since(started).Round(time.Millisecond), err)
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	var (
		text        string
		contentType string
	)
	if format == "original" {
		// Original format (srt/ass/ssa/vtt) without conversion; only the encoding changes
		// (the source encoding is detected automatically and converted to the requested one).
		text = subs.Decode(raw)
		contentType = "text/plain"
	} else {
		text, err = subs.ToVTT(raw, track.Format)
		if err != nil {
			log.Printf("subs: convert file %d (%s) failed: %v", track.FileIndex, track.Name, err)
			c.AbortWithError(http.StatusUnprocessableEntity, err)
			return
		}
		contentType = "text/vtt"
	}

	out, charset, err := subs.Encode(text, encoding)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// CORS headers are not set here on purpose: the global CORS middleware already adds
	// Access-Control-Allow-Origin (a duplicated header breaks browsers).
	c.Data(http.StatusOK, contentType+"; charset="+charset, out)
}
