//go:build !slim

package pages

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"server/settings"
	"server/web/auth"
)

// lockedEngine: --httpauth with accs.db {"user":"secret"} and the web UI routes of this build.
func lockedEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldAuth, oldPath := settings.HttpAuth, settings.Path
	settings.Path, settings.HttpAuth = t.TempDir(), true
	t.Cleanup(func() { settings.HttpAuth, settings.Path = oldAuth, oldPath })
	if err := os.WriteFile(filepath.Join(settings.Path, "accs.db"), []byte(`{"user":"secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	e := gin.New()
	auth.SetupAuth(e)
	SetupRoute(e)
	return e
}

func get(e *gin.Engine, path string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w
}
