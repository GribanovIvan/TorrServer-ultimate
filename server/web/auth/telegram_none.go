//go:build !full

package auth

import "github.com/gin-gonic/gin"

// Telegram Mini App login (X-Telegram-Init-Data) is in the full profile only.
func telegramUser(*gin.Context) (string, bool) { return "", false }
