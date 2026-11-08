//go:build full

package api

import (
	"github.com/gin-gonic/gin"

	sets "server/settings"
)

// Routes only the full profile serves.
func setupFullRoutes(route gin.IRouter, authorized *gin.RouterGroup) {
	// External audio tracks of a video (sirin-media/TorrServer-NG). Like
	// /subs it checks auth itself: players fetch it without credentials.
	route.GET("/audio/:hash/:id", audioTracks)

	if sets.Args != nil && sets.Args.Pprof {
		mountPprof(authorized)
	}
}

// shutdownEnabled is false with --disable-shutdown, for servers shared with
// people who must not stop them (idea from reyohoho/TorrServer, 1496fa0a).
func shutdownEnabled() bool {
	return sets.Args == nil || !sets.Args.DisableShutdown
}
