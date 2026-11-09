//go:build !slim && newui

package pages

import (
	"github.com/gin-gonic/gin"

	"server/web/auth"
	"server/web/pages/template"
)

func SetupRoute(route gin.IRouter) {
	authorized := route.Group("/", auth.CheckAuth())

	// SPA + static assets are public so the web UI can show a custom Basic login
	// form. API routes (and /stat, /magnets) stay behind CheckAuth.
	template.RouteWebPages(route)
	authorized.GET("/stat", statPage)
	authorized.GET("/magnets", getTorrents)
}
