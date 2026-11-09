//go:build !slim && !newui

package pages

import (
	"slices"

	"github.com/gin-gonic/gin"

	"server/web/auth"
	"server/web/pages/template"
)

// SetupRoute of the classic web UI, as upstream MatriX.150: the pages themselves are behind
// CheckAuth (the browser asks for the password with its own Basic dialog), except the manifest.
func SetupRoute(route gin.IRouter) {
	authorized := route.Group("/", auth.CheckAuth())

	webPagesAuth := route.Group("/", func() gin.HandlerFunc {
		return func(c *gin.Context) {
			if slices.Contains([]string{"/site.webmanifest"}, c.FullPath()) {
				return
			}
			auth.CheckAuth()(c)
		}
	}())

	template.RouteWebPages(webPagesAuth)
	authorized.GET("/stat", statPage)
	authorized.GET("/magnets", getTorrents)
}
