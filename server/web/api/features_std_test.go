//go:build !slim

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	sets "server/settings"
	"server/torr/offlinetest"
)

var testSubsHandler gin.HandlerFunc = subsTracks

// restartServices applies settings to DLNA, Bonjour and rutor. With all three
// off it only stops them; it must not bind a port or leave a goroutine.
func TestRestartServicesWithEverythingOff(t *testing.T) {
	offlinetest.Prepare(t)
	sets.BTsets.EnableDLNA = false
	sets.BTsets.EnableBonjour = false
	sets.BTsets.EnableRutorSearch = false
	restartServices()
}

func TestTMDBSettingsUninitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := sets.BTsets
	sets.BTsets = nil
	t.Cleanup(func() { sets.BTsets = old })

	r := gin.New()
	r.GET("/tmdb/settings", tmdbSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tmdb/settings", nil))
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "Settings not initialized") {
		t.Fatalf("tmdb %d %s", w.Code, w.Body.String())
	}
}
