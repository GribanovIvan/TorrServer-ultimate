//go:build gst && !full && !slim

package gstreamer

import "github.com/gin-gonic/gin"

func rememberRequestUser(*gin.Context) {}
