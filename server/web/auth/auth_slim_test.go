//go:build slim

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"server/settings"
)

// Without a web UI every unauthenticated request gets the Basic challenge, also
// from SPA-like clients, and Telegram Mini App data is not accepted.
func TestCheckAuthAlwaysChallenges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings.HttpAuth = true
	t.Cleanup(func() { settings.HttpAuth = false })

	r := gin.New()
	r.GET("/probe", CheckAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, h := range []map[string]string{
		{},
		{"Accept": "application/json"},
		{"X-Requested-With": "XMLHttpRequest"},
		{"Sec-Fetch-Mode": "cors"},
		{"X-Telegram-Init-Data": "user=%7B%22id%22%3A1%7D&hash=00"},
	} {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		for k, v := range h {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("%v: %d %q", h, w.Code, w.Header().Get("WWW-Authenticate"))
		}
	}
}
