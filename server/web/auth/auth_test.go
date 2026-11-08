package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"server/settings"
)

func TestSetupAuthMissingAndValidAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldAuth, oldPath := settings.HttpAuth, settings.Path
	settings.Path = t.TempDir()
	settings.HttpAuth = false
	t.Cleanup(func() {
		settings.HttpAuth = oldAuth
		settings.Path = oldPath
	})

	open := gin.New()
	SetupAuth(open)
	open.GET("/open", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	open.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/open", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("auth off %d", w.Code)
	}

	settings.HttpAuth = true
	missing := gin.New()
	SetupAuth(missing)
	missing.GET("/open", func(c *gin.Context) { c.Status(http.StatusOK) })
	w = httptest.NewRecorder()
	missing.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/open", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("missing accs.db %d", w.Code)
	}

	accs := filepath.Join(settings.Path, "accs.db")
	if err := os.WriteFile(accs, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := gin.New()
	SetupAuth(bad)

	if err := os.WriteFile(accs, []byte(`{"user":"secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := gin.New()
	SetupAuth(locked)
	locked.GET("/open", CheckAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w = httptest.NewRecorder()
	locked.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/open", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no credentials %d", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/open", nil)
	req.SetBasicAuth("user", "secret")
	w = httptest.NewRecorder()
	locked.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("basic auth %d", w.Code)
	}
}

func TestCheckAuthAllowsAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings.HttpAuth = true
	t.Cleanup(func() { settings.HttpAuth = false })

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(gin.AuthUserKey, "alice")
		c.Next()
	})
	r.GET("/probe", CheckAuth(), func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
}
