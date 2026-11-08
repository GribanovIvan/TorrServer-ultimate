//go:build slim

package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// The slim profile serves the API over plain HTTP only: no web UI, HTTPS (a
// reverse proxy terminates TLS), IP blocklists, MSX, MCP, DLNA, Bonjour,
// GStreamer, FUSE, WebDAV, rutor or Swagger, and no internal loopback listener
// (only ffprobe and GStreamer use it). These hooks stay empty.

var featureHeaders []string

func prepareTLS() error                    { return nil }
func startServices()                       {}
func firewall() []gin.HandlerFunc          { return nil }
func setupFeatures(*gin.Engine)            {}
func stopFeatures()                        {}
func startTLS(http.Handler) error          { return nil }
func startInternalServer(http.Handler)     {}
func stopTLS()                             {}
func warnHTTPSOnly()                       {}
func forceHTTPS(http.Handler) http.Handler { return nil }
