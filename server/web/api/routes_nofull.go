//go:build !full

package api

import "github.com/gin-gonic/gin"

func setupFullRoutes(gin.IRouter, *gin.RouterGroup) {}

func shutdownEnabled() bool { return true }
