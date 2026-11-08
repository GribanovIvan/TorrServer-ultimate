package version

import (
	"log"
	"runtime/debug"
)

// Version is set at build time via -ldflags "-X server/version.Version=..." (GoReleaser {{.Version}}).
// The number follows the NextGen base this branch is built on.
var Version = "Ultimate.150"

// Full is Version with the build profile appended ("Ultimate.150-slim"); the
// default profile has no suffix. It is what --version and the start log show.
func Full() string {
	if Profile == "" {
		return Version
	}
	return Version + "-" + Profile
}

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
