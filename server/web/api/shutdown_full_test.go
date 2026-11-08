//go:build full

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	sets "server/settings"
)

func TestDisableShutdownDropsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := sets.Args
	t.Cleanup(func() { sets.Args = old })
	sets.Args = &sets.ExecArgs{DisableShutdown: true}
	r := gin.New()
	SetupRoute(r)
	for _, p := range []string{"/shutdown", "/shutdown/x"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d, want 404", p, w.Code)
		}
	}
}
