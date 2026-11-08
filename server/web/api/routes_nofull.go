//go:build !full

package api

import "github.com/gin-gonic/gin"

func setupFullRoutes(*gin.RouterGroup) {}
