//go:build full

package api

import (
	"github.com/gin-gonic/gin"

	sets "server/settings"
)

// Routes only the full profile serves.
func setupFullRoutes(authorized *gin.RouterGroup) {
	if sets.Args != nil && sets.Args.Pprof {
		mountPprof(authorized)
	}
}

// shutdownEnabled is false with --disable-shutdown, for servers shared with
// people who must not stop them (idea from reyohoho/TorrServer, 1496fa0a).
func shutdownEnabled() bool {
	return sets.Args == nil || !sets.Args.DisableShutdown
}
