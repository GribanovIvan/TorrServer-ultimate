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
