//go:build !slim && newui

package api

import (
	"github.com/gin-gonic/gin"

	sets "server/settings"
)

// API that only the NextGen web UI (-tags newui) uses: indexer capabilities for its torznab search
// form and the runtime status panel. Upstream MatriX.150, and so the default profile with the classic
// UI, has neither.
// newUIRoutes: this build registers the NextGen UI API (tests check both builds).
const newUIRoutes = true

func setupNewUIRoutes(route gin.IRouter, authorized *gin.RouterGroup) {
	if sets.SearchWA {
		route.GET("/torznab/caps", torznabCaps)
	} else {
		authorized.GET("/torznab/caps", torznabCaps)
	}
	authorized.GET("/runtime/status", runtimeStatus)
}
