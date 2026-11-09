//go:build !slim && !newui

package auth

import "github.com/gin-gonic/gin"

// Classic web UI (default profile), as upstream MatriX.150: every unauthorized request gets the Basic
// challenge, the browser shows its own password dialog. The NextGen UI (-tags newui) has its own login
// form and suppresses the challenge for its XHR calls (auth_std.go).
func shouldSendBasicChallenge(*gin.Context) bool { return true }
