//go:build !slim

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/browser"

	"server/docs"
	"server/log"
	"server/settings"
	"server/torr"
	"server/torr/utils"
	"server/version"
)

// featureArgs are the flags of the default and full profiles beyond the core in
// main.go. The slim profile replaces this file with args_slim.go.
type featureArgs struct {
	Ssl         bool   `help:"enables https"`
	SslPort     string `help:"web server ssl port, If not set, will be set to default 8091 or taken from db(if stored previously). Accepted if --ssl enabled."`
	SslCert     string `help:"path to ssl cert file. If not set, will be taken from db(if stored previously) or default self-signed certificate/key will be generated. Accepted if --ssl enabled."`
	SslKey      string `help:"path to ssl key file. If not set, will be taken from db(if stored previously) or default self-signed certificate/key will be generated. Accepted if --ssl enabled."`
	UI          bool   `arg:"-u" help:"open torrserver page in browser"`
	TorrentsDir string `arg:"-t" help:"autoload torrents from dir"`
	SearchWA    bool   `arg:"-s" help:"search without auth"`
	TGToken     string `arg:"-T" help:"telegram bot token"`
	FusePath    string `arg:"-f" help:"fuse mount path"`
	WebDAV      bool   `help:"web dav enable"`
	ForceHTTPS  bool   `arg:"--force-https" help:"redirect all HTTP requests to HTTPS (requires --ssl)"`
	HTTPSOnly   bool   `arg:"--https-only" help:"with --ssl, don't open the plain HTTP port (--port): everything is served over HTTPS only. Players and TVs need a trusted certificate"`
	HTTPMedia   bool   `arg:"--http-media" help:"with --force-https, keep serving media URLs (/stream, /play, playlists) over plain HTTP for players and TVs that can't use HTTPS. Stream URLs and Basic auth credentials then travel unencrypted: use on trusted networks only"`
	Pprof       bool   `arg:"--pprof" help:"enable /debug/pprof (pair with -a; exposes heap)"`
	DownloadDir string `arg:"-x,--downloaddir" help:"serve files already downloaded here (e.g. by qBittorrent) instead of streaming them; the path is <dir>/<torrent file path>, and only a file of the exact size is used, so keep incomplete files out of it"`
}

func (f featureArgs) apply(a *settings.ExecArgs) {
	a.Ssl = f.Ssl
	a.SslPort = f.SslPort
	a.SslCert = f.SslCert
	a.SslKey = f.SslKey
	a.UI = f.UI
	a.TorrentsDir = f.TorrentsDir
	a.SearchWA = f.SearchWA
	a.TGToken = f.TGToken
	a.FusePath = f.FusePath
	a.WebDAV = f.WebDAV
	a.ForceHTTPS = f.ForceHTTPS
	a.HTTPMedia = f.HTTPMedia
	a.HTTPSOnly = f.HTTPSOnly
	a.Pprof = f.Pprof
	settings.DownloadDir = f.DownloadDir
}

func (f featureArgs) check() error {
	switch {
	case f.ForceHTTPS && !f.Ssl:
		return errors.New("--force-https requires --ssl")
	case f.HTTPSOnly && !f.Ssl:
		return errors.New("--https-only requires --ssl")
	case f.HTTPSOnly && f.HTTPMedia:
		return errors.New("--https-only and --http-media can't be combined (--https-only doesn't open the HTTP port)")
	case f.HTTPMedia && !f.ForceHTTPS:
		return errors.New("--http-media requires --force-https (without it, media is already served over HTTP)")
	}
	return nil
}

// start runs what the flags ask for once they passed check.
func (f featureArgs) start(port string) {
	docs.SwaggerInfo.Version = version.Version

	if f.UI {
		go func() {
			time.Sleep(time.Second)
			if f.Ssl {
				_ = browser.OpenURL("https://127.0.0.1:" + f.SslPort)
			} else {
				_ = browser.OpenURL("http://127.0.0.1:" + port)
			}
		}()
	}

	if f.TorrentsDir != "" {
		go watchTDir(f.TorrentsDir)
	}

	if f.Pprof {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(5)
		log.TLogln("pprof enabled at /debug/pprof/")
	}
}

func watchTDir(dir string) { watchDir(dir, nil) }

func watchDir(dir string, stop <-chan struct{}) {
	path, err := filepath.Abs(dir) // Attempt to convert the provided dir path into an absolute (full) filesystem path.
	if err != nil {
		path = dir
	} // If an error occurs while obtaining the absolute path, the original relative path is used.

	watcher, err := fsnotify.NewWatcher() // Create a new filesystem watcher (event-based instead of polling).
	if err != nil {
		log.TLogln("Error creating watcher:", err)
		return
	}
	defer func() { _ = watcher.Close() }()

	err = watcher.Add(path) // Add target directory to watcher to receive filesystem events.
	if err != nil {
		log.TLogln("Error adding directory to watcher:", err)
		return
	}

	for { // Start of an infinite loop for continuous background monitoring using filesystem events.

		select {
		case <-stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Process only file creation or modification events
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}

			func() { // Open an anonymous function to isolate processing context per file.
				defer func() { // Deferred execution of a safety function that always runs at the end (even in case of panic).
					if r := recover(); r != nil { // Catch critical runtime errors (panic) during torrent processing.
						log.TLogln("Recovered from panic in watchTDir:", r) // Log recovered panic to prevent full service crash.
					}
				}()

				filename := event.Name                                     // Retrieves the full path of the file directly from the event generated by the watcher.
				if strings.ToLower(filepath.Ext(filename)) != ".torrent" { // Extract the file extension, convert it to lowercase, and check it. If it is not .torrent, the file is ignored.
					return // Ends execution of the anonymous function for the current file and moves to the next event.
				}

				sp, err := utils.OpenTorrentFile(filename) // Parse the torrent file: read Bencode structure and extract metadata.
				if err != nil {
					log.TLogln("Error parse file name:", err)
					return
				}

				tor, err := torr.AddTorrent(sp, "", "", "", "") // Add parsed torrent specification into active torrent engine.
				if err != nil {
					log.TLogln("Error parse torrent file:", err)
					return
				}

				if !tor.GotInfo() { // Check whether torrent metadata (info section) is available.
					log.TLogln("Error get info from torrent") // If metadata is missing, log error and skip processing.
					return
				}

				// If torrent has no custom title, assign default name from metadata.
				tor.EnsureTitle(func() string { return tor.Name() })

				// Long database operation is now safe
				torr.SaveTorrentToDB(tor) // Save torrent metadata into SQLite database.
				tor.Drop()                // Free RAM by removing torrent from active memory after processing.

				// Safely remove the original torrent file before writing to database.
				// This prevents reprocessing in case of repeated filesystem events.
				if err := os.Remove(filename); err != nil {
					log.TLogln("Error removing torrent file:", err)
				}
			}()

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Log filesystem watcher errors
			log.TLogln("Watcher error:", err)
		}
	}
}
