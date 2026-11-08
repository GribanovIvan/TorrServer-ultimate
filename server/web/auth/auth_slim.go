//go:build slim

package auth

import "github.com/gin-gonic/gin"

// The slim profile has no web UI login form: only Basic auth from accs.db,
// always with the WWW-Authenticate challenge.

func shouldSendBasicChallenge(*gin.Context) bool { return true }
