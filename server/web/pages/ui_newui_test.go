//go:build !slim && newui

package pages

import (
	"net/http"
	"strings"
	"testing"
)

// -tags newui: the NextGen SPA is public (it shows its own login form); its XHR to the API gets 401
// without the Basic challenge, so the browser does not pop up its dialog.
func TestNewUIPublicWithOwnLogin(t *testing.T) {
	e := lockedEngine(t)
	w := get(e, "/", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<title>TorrServer NextGen</title>") {
		t.Fatalf("/ %d", w.Code)
	}
	w = get(e, "/stat", map[string]string{"Accept": "application/json"})
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("/stat from the SPA: %d, WWW-Authenticate %q", w.Code, w.Header().Get("WWW-Authenticate"))
	}
}
