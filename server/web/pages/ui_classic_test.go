//go:build !slim && !newui

package pages

import (
	"net/http"
	"strings"
	"testing"
)

// Default profile: the classic MatriX UI, as upstream — the pages are behind Basic auth with the
// browser's own dialog (always WWW-Authenticate), only the manifest is public.
func TestClassicUIBehindBasicAuth(t *testing.T) {
	e := lockedEngine(t)
	for _, h := range []map[string]string{nil, {"Accept": "application/json"}} {
		w := get(e, "/", h)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("/ without password %v: %d, WWW-Authenticate %q", h, w.Code, w.Header().Get("WWW-Authenticate"))
		}
	}
	if w := get(e, "/site.webmanifest", nil); w.Code != http.StatusOK {
		t.Fatalf("manifest %d", w.Code)
	}
	w := get(e, "/", map[string]string{"Authorization": "Basic dXNlcjpzZWNyZXQ="})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<title>TorrServer MatriX</title>") {
		t.Fatalf("/ with password: %d", w.Code)
	}
	// Our copy of the upstream page has the Firebase Analytics of the upstream project cut out.
	if strings.Contains(w.Body.String(), "firebase") {
		t.Fatal("classic index.html still loads Firebase Analytics")
	}
}
