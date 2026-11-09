//go:build !slim && newui

package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SPA XHR/fetch probes must not trigger the browser's native Basic dialog, so
// the web UI can show its own login form. Third-party clients that already send
// Authorization never hit this path.
// shouldSendBasicChallenge is false for typical SPA/API clients (JSON Accept,
// X-Requested-With, or Sec-Fetch-Mode: cors) so axios can show a custom login UI.
func shouldSendBasicChallenge(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	if strings.Contains(accept, "application/json") || strings.Contains(accept, "+json") {
		return false
	}
	if c.GetHeader("X-Requested-With") != "" {
		return false
	}
	if strings.EqualFold(c.GetHeader("Sec-Fetch-Mode"), "cors") {
		return false
	}
	return true
}
