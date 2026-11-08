//go:build !full

package api

import (
	"github.com/gin-gonic/gin"

	sets "server/settings"
	"server/torr/state"
)

// Without per-user data (full profile, peruser.go) the hooks in the handlers
// change nothing.

const (
	lastViewed  = ""
	lastPreload = ""
)

func currentUser(*gin.Context) string { return "" }

func requestUser(*gin.Context, string, string) string { return "" }

func userParam(string) string { return "" }

func settingsForUser(*gin.Context) any { return sets.BTsets }

func viewedStatus(tor *state.TorrentStatus, _ string) *state.TorrentStatus { return tor }
