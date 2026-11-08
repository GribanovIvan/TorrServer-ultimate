//go:build !slim

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexflint/go-arg"

	"server/torr/offlinetest"
)

func watchDirArgs(dir string) []string { return []string{"-t", dir} }

// checkWatchDirAddsTorrent drops a .torrent into the -t dir and waits for the
// watcher to take it.
func checkWatchDirAddsTorrent(t *testing.T, dir string) {
	show := offlinetest.WriteShow(t)
	var meta bytes.Buffer
	if err := show.Meta.Write(&meta); err != nil {
		t.Fatal(err)
	}
	torrentPath := filepath.Join(dir, "show.torrent")
	if err := os.WriteFile(torrentPath, meta.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(torrentPath); err == nil {
		t.Fatal("watcher left the torrent file")
	}
}

func TestFeatureArgs(t *testing.T) {
	var params args
	p, err := arg.NewParser(arg.Config{}, &params)
	if err != nil {
		t.Fatal(err)
	}
	err = p.Parse([]string{
		"--ssl",
		"--sslport", "9443",
		"--https-only",
		"--force-https",
		"--http-media",
		"--pprof",
	})
	if err != nil {
		t.Fatal(err)
	}
	if params.SslPort != "9443" || !params.Ssl || !params.HTTPSOnly || !params.ForceHTTPS || !params.HTTPMedia || !params.Pprof {
		t.Fatalf("%+v", params)
	}
}

func TestRunRejectsBadHTTPSFlags(t *testing.T) {
	dir := t.TempDir()
	base := []string{"-k", "-a", "-r", "-d", dir, "-p", "1", "-t", dir, "-m", "1", "-4", "8.8.8.8", "-6", "::1", "--torrentaddr", "127.0.0.1:0", "--proxyurl", "socks5://127.0.0.1:1", "--proxymode", "nope", "--pprof"}
	if code := run(append(append([]string{}, base...), "--force-https")); code != 1 {
		t.Fatal("force-https", code)
	}
	if code := run([]string{"-k", "-d", dir, "--https-only"}); code != 1 {
		t.Fatal("https-only", code)
	}
	if code := run([]string{"-k", "-d", dir, "--ssl", "--https-only", "--http-media"}); code != 1 {
		t.Fatal("combined", code)
	}
	if code := run([]string{"-k", "-d", dir, "--http-media"}); code != 1 {
		t.Fatal("http-media", code)
	}
}
