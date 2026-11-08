//go:build slim

package web

import (
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"server/settings"
	"server/version"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ts-slim-web")
	if err != nil {
		panic(err)
	}
	settings.Path = dir
	if err := settings.InitSets(false, false); err != nil {
		panic(err)
	}
	code := m.Run()
	settings.CloseDB()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// TestSlimServesCoreOnly starts the slim server and checks that the core API
// answers while the routes of features left out of the profile are 404.
func TestSlimServesCoreOnly(t *testing.T) {
	settings.BTsets = &settings.BTSets{
		CacheSize: 32 << 20, PreloadCache: 100, ReaderReadAHead: 50, ConnectionsLimit: 25,
		TorrentDisconnectTimeout: 120, ResponsiveMode: true,
		DisableDHT: true, DisablePEX: true, DisableUPNP: true, DisableUTP: true,
	}
	settings.PubIPv4, settings.TorAddr = "8.8.8.8", "127.0.0.1:0"
	settings.Args = &settings.ExecArgs{}
	settings.IPs = []string{"127.0.0.1"}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, settings.Port, _ = net.SplitHostPort(ln.Addr().String())
	_ = ln.Close()

	if err := Start(); err != nil {
		t.Fatal(err)
	}
	base := "http://127.0.0.1:" + settings.Port
	call := func(method, path, body string) (int, string) {
		t.Helper()
		req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, body := call("GET", "/echo", ""); code != 200 || body != version.Version {
		t.Fatalf("/echo: %d %q", code, body)
	}
	if settings.InternalPort != "" {
		t.Fatalf("slim opened the internal loopback listener on %s", settings.InternalPort)
	}
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/torrents", `{"action":"list"}`},
		{"POST", "/settings", `{"action":"get"}`},
		{"POST", "/viewed", `{"action":"list"}`},
		{"GET", "/playlistall/all.m3u", ""},
	} {
		if code, body := call(c.method, c.path, c.body); code != 200 {
			t.Fatalf("%s %s: %d %s", c.method, c.path, code, body)
		}
	}
	for _, p := range []string{
		"/", "/index.html", "/swagger/index.html", "/stat", "/magnets",
		"/msx/start.json", "/mcp", "/ssl/status", "/dav/", "/waf", "/mediabase",
		"/search/x", "/torznab/caps", "/torznab/search/x", "/subs/0/1",
		"/storage/settings", "/tmdb/settings", "/gst/settings", "/ffp/status",
		"/runtime/status", "/debug/pprof/",
	} {
		if code, _ := call("GET", p, ""); code != http.StatusNotFound {
			t.Fatalf("GET %s: %d, want 404", p, code)
		}
	}

	waited := make(chan error, 1)
	go func() { waited <- Wait() }()
	go Stop()
	select {
	case err := <-waited:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Wait did not return after Stop")
	}
}
