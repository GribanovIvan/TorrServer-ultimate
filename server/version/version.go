package version

import (
	"log"
	"runtime/debug"
)

// Version can be overridden at build time via -ldflags "-X server/version.Version=<tag>"
// Client apps detect server capabilities by finding "MatriX" in this string
// and reading the first number; without the token they fall back to a legacy
// protocol mode. Keep the "MatriX.<n>" shape in any rename.
var Version = "MatriX.200-Tommix"

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
