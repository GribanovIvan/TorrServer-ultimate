//go:build full

package main

import (
	"runtime"

	"server/log"
	"server/settings"
)

// fullArgs are the flags only the full profile has.
type fullArgs struct {
	Pprof       bool   `arg:"--pprof" help:"enable /debug/pprof (pair with -a; exposes heap)"`
	DownloadDir string `arg:"-x,--downloaddir" help:"serve files already downloaded here (e.g. by qBittorrent) instead of streaming them; the path is <dir>/<torrent file path>, and only a file of the exact size is used, so keep incomplete files out of it"`
}

func (f fullArgs) apply(a *settings.ExecArgs) {
	a.Pprof = f.Pprof
	settings.DownloadDir = f.DownloadDir
}

func (f fullArgs) start() {
	if f.Pprof {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(5)
		log.TLogln("pprof enabled at /debug/pprof/")
	}
}
