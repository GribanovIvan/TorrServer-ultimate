//go:build !slim

package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"server/bonjour"
	"server/dlna"
	gstreamer "server/gstreamer/bridge"
	"server/rutor"
	sets "server/settings"
	"server/web/sslcerts"
)

// API of the default and full profiles beyond the core in route.go. The slim
// profile replaces this file with routes_slim.go.

func setupFeatureRoutes(route gin.IRouter, authorized *gin.RouterGroup) {
	authorized.GET("/waf", getWAF)
	authorized.GET("/mediabase", mediaBase)
	authorized.POST("/waf", updateWAF)
	authorized.POST("/torznab/test", torznabTest)

	route.GET("/subs/:hash/:id", subsTracks)

	if sets.SearchWA {
		route.GET("/search/*query", rutorSearch)
	} else {
		authorized.GET("/search/*query", rutorSearch)
	}

	if sets.SearchWA {
		route.GET("/torznab/search/*query", torznabSearch)
		route.GET("/torznab/caps", torznabCaps)
	} else {
		authorized.GET("/torznab/search/*query", torznabSearch)
		authorized.GET("/torznab/caps", torznabCaps)
	}
	// Storage settings endpoints
	authorized.GET("/storage/settings", GetStorageSettings)
	authorized.POST("/storage/settings", UpdateStorageSettings)
	// TMDB settings endpoint
	authorized.GET("/tmdb/settings", tmdbSettings)
	// GStreamer
	authorized.GET("/gst/settings", GetGStreamerSettings)
	authorized.POST("/gst/settings", UpdateGStreamerSettings)
	// FFProbe
	authorized.GET("/ffp/status", ffprobeStatus)
	authorized.GET("/ffp/:hash/:id", ffp)

	authorized.GET("/runtime/status", runtimeStatus)

	if sets.Args != nil && sets.Args.Pprof {
		mountPprof(authorized)
	}
}

// restartServices applies the settings now in effect to DLNA, Bonjour and the
// rutor search.
func restartServices() {
	dlna.Stop()
	if sets.BTsets.EnableDLNA {
		dlna.Start()
	}
	bonjour.Stop()
	if sets.BTsets.EnableBonjour {
		bonjour.Start()
	}
	rutor.Stop()
	rutor.Start()
}

// refreshDLNA rebuilds the DLNA library after the torrent list changed.
func refreshDLNA() {
	if sets.BTsets.EnableDLNA {
		dlna.Stop()
		dlna.Start()
	}
}

// dropTranscodes stops GStreamer sessions of a removed torrent.
func dropTranscodes(hash string) {
	_ = gstreamer.Remove(hash)
}

// plainMediaBase points media links at the plain HTTP port when the request came
// over TorrServer's self-signed certificate, which players reject, and that port
// serves media.
func plainMediaBase(c *gin.Context) (string, bool) {
	if c.Request.TLS != nil && sets.BTsets != nil && sets.Port != "" && sets.PlainHTTPServesMedia() &&
		sslcerts.IsGenerated(sets.BTsets.SslCert, sets.BTsets.SslKey) {
		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = strings.TrimSuffix(strings.TrimPrefix(c.Request.Host, "["), "]")
		}
		return "http://" + net.JoinHostPort(host, sets.Port), true
	}
	return "", false
}

type mediaBaseResponse struct {
	Base string `json:"base"`
}

// mediaBase godoc
//
//	@Summary		Base URL for external players
//	@Description	Scheme and host that links handed to external players (VLC, copied links)
//	@Description	should use. It differs from the web UI origin when the UI is served over
//	@Description	TorrServer's self-signed HTTPS certificate, which players reject, and the
//	@Description	plain HTTP port serves media. In-page playback should keep using the UI origin.
//
//	@Tags			API
//
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	mediaBaseResponse
//	@Router			/mediabase [get]
func mediaBase(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, mediaBaseResponse{Base: mediaBaseURL(c)})
}
