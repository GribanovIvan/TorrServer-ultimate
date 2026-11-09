//go:build !slim && !newui

package api

import "github.com/gin-gonic/gin"

// The classic web UI needs no API beyond upstream's; see routes_newui.go.
const newUIRoutes = false

func setupNewUIRoutes(gin.IRouter, *gin.RouterGroup) {}
