package api

import (
	"server/web/auth"

	"github.com/gin-gonic/gin"
)

type requestI struct {
	Action string `json:"action,omitempty"`
}

// SetupRoute registers the API every profile serves: torrents, streaming,
// playlists, settings and viewed marks. Routes of other features come from
// setupFeatureRoutes (routes_std.go, empty in the slim profile).
func SetupRoute(route gin.IRouter) {
	authorized := route.Group("/", auth.CheckAuth())

	authorized.GET("/shutdown", shutdown)
	authorized.GET("/shutdown/*reason", shutdown)

	authorized.POST("/settings", settings)

	authorized.POST("/torrents", torrents)

	authorized.POST("/torrent/upload", torrentUpload)

	authorized.POST("/cache", cache)

	route.HEAD("/stream", stream)
	route.GET("/stream", stream)

	route.HEAD("/stream/*fname", stream)
	route.GET("/stream/*fname", stream)

	route.HEAD("/play/:hash/:id", play)
	route.GET("/play/:hash/:id", play)

	authorized.POST("/viewed", viewed)

	authorized.GET("/playlistall/all.m3u", allPlayList)

	route.GET("/playlist", playList)
	route.GET("/playlist/*fname", playList)

	authorized.GET("/download/:size", download)

	setupFeatureRoutes(route, authorized)
}
