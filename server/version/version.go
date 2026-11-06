package version

import (
	"log"
	"runtime/debug"
)

// Version is set at build time via -ldflags "-X server/version.Version=<tag>"
var Version = "NG-1.0.0-dev (based on MatriX.149)"

func GetTorrentVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Printf("Failed to read build info")
		return ""
	}
	for _, dep := range bi.Deps {
		if dep.Path == "github.com/anacrolix/torrent" {
			if dep.Replace != nil {
				return dep.Replace.Version
			}

			return dep.Version
		}
	}
	return ""
}
