//go:build full

package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"server/settings"
)

func TestAccountsReloadOnChange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldPath, oldAuth := settings.Path, settings.HttpAuth
	t.Cleanup(func() { settings.Path, settings.HttpAuth = oldPath, oldAuth })
	accsPollInterval = 20 * time.Millisecond // read once, when the watcher starts
	settings.Path = t.TempDir()
	settings.HttpAuth = true
	accs := filepath.Join(settings.Path, "accs.db")
	if err := os.WriteFile(accs, []byte(`{"alice":"one"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	SetupAuth(r)
	r.GET("/p", CheckAuth(), func(c *gin.Context) { c.String(http.StatusOK, c.GetString(gin.AuthUserKey)) })
	status := func(user, pass string) int {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	waitFor := func(what string, ok func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for !ok() {
			if time.Now().After(deadline) {
				t.Fatal(what)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if status("alice", "one") != http.StatusOK || status("bob", "two") != http.StatusUnauthorized {
		t.Fatal("initial accounts")
	}
	if err := os.WriteFile(accs, []byte(`{"bob":"two"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor("new account not picked up", func() bool { return status("bob", "two") == http.StatusOK })
	if status("alice", "one") != http.StatusUnauthorized {
		t.Fatal("removed account still accepted")
	}
	// A broken file keeps the accounts in force.
	if err := os.WriteFile(accs, []byte(`{"bob":`), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if status("bob", "two") != http.StatusOK {
		t.Fatal("broken accs.db dropped the accounts")
	}
}
