//go:build slim

package api

import "github.com/gin-gonic/gin"

// The slim profile serves only the core API of route.go: no web UI helpers,
// blocklists, search, subtitles, ffprobe, GStreamer, DLNA or Bonjour. These
// hooks stay empty.

func setupFeatureRoutes(gin.IRouter, *gin.RouterGroup) {}
func restartServices()                                 {}
func refreshDLNA()                                     {}
func dropTranscodes(string)                            {}
func plainMediaBase(*gin.Context) (string, bool)       { return "", false }
