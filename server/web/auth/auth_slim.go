//go:build slim

package auth

import "github.com/gin-gonic/gin"

// The slim profile has no web UI login form and no Telegram Mini App: only
// Basic auth from accs.db, always with the WWW-Authenticate challenge.

func telegramUser(*gin.Context) (string, bool) { return "", false }

func shouldSendBasicChallenge(*gin.Context) bool { return true }
