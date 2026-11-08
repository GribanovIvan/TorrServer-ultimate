//go:build windows && !slim

package fuse

import (
	"server/log"
	"server/settings"
)

func FuseAutoMount() {
	if settings.Args.FusePath != "" {
		log.TLogln("Windows not support FUSE")
	}
}

func FuseCleanup() {
}
