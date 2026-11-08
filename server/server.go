package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"server/log"
	"server/netbind"
	"server/settings"
	"server/torr/utils"
	"server/web"
)

var stopOnce sync.Once

func Start() error {
	stopOnce = sync.Once{}
	settings.EmbeddedStop = Stop

	if err := settings.InitSets(settings.Args.RDB, settings.Args.SearchWA); err != nil {
		return err
	}
	if err := prepareSSL(); err != nil {
		return err
	}
	// http checks
	if settings.Args.Port == "" {
		settings.Args.Port = settings.DefaultPort
	}

	// --https-only never opens the HTTP port, so it doesn't have to be free
	if settings.HTTPEnabled() {
		log.TLogln("Check web port", settings.Args.Port, "on", netbind.Normalize(settings.Args.IPs))
		if err := netbind.CheckPort(settings.Args.IPs, settings.Args.Port); err != nil {
			msg := fmt.Sprintf("cannot bind HTTP port %s: %v", settings.Args.Port, err)
			log.TLogln(msg)
			return errors.New(msg)
		}
	}
	// remove old disk caches
	go cleanCache()
	// set settings http and https ports. Start web server.
	settings.Port = settings.Args.Port
	settings.SslPort = settings.Args.SslPort
	settings.IPs = settings.Args.IPs

	startTelegramBot()
	return web.Start()
}

func cleanCache() {
	if !settings.DiskCache || !settings.BTsets.UseDisk || settings.BTsets.TorrentsSavePath == "/" || settings.BTsets.TorrentsSavePath == "" {
		return
	}

	dirs, err := os.ReadDir(settings.BTsets.TorrentsSavePath)
	if err != nil {
		return
	}

	torrs := settings.ListTorrent()

	log.TLogln("Remove unused cache in dir:", settings.BTsets.TorrentsSavePath)
	keep := map[string]bool{}
	for _, d := range dirs {
		if len(d.Name()) != 40 {
			// Not a hash
			continue
		}

		if !settings.BTsets.RemoveCacheOnDrop {
			keep[d.Name()] = true
			for _, t := range torrs {
				if d.IsDir() && d.Name() == t.InfoHash.HexString() {
					keep[d.Name()] = false
					break
				}
			}
			for hash, del := range keep {
				if del && hash == d.Name() {
					log.TLogln("Remove unused cache:", d.Name())
					removeAllFiles(filepath.Join(settings.BTsets.TorrentsSavePath, d.Name()))
				}
			}
		} else {
			if d.IsDir() {
				log.TLogln("Remove unused cache:", d.Name())
				removeAllFiles(filepath.Join(settings.BTsets.TorrentsSavePath, d.Name()))
			}
		}
	}
}

func removeAllFiles(path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		name := filepath.Join(path, f.Name())
		_ = os.Remove(name)
	}
	_ = os.Remove(path)
}

func WaitServer() string {
	err := web.Wait()
	if err != nil {
		return err.Error()
	}
	return ""
}

func Stop() {
	stopOnce.Do(func() {
		web.Stop()
		settings.CloseDB()
	})
}

func AddTrackers(trackers string) {
	lines := strings.Split(trackers, "\n")
	var tracks []string
	for _, l := range lines {
		l = strings.Trim(l, " ,\r")
		if l != "" {
			tracks = append(tracks, l)
		}
	}
	utils.SetDefTrackers(tracks)
}
