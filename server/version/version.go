package version

import (
	"log"
	"runtime/debug"
)

// Version is set at build time via -ldflags "-X server/version.Version=..." (GoReleaser {{.Version}}).
// The number is the upstream MatriX release whose code the branch contains (MatriX.150: everything up to
// it came in through lampac-nextgen, MatriX.150 itself only bumped the label), not the NextGen label.
var Version = "Ultimate.150"

// Full is Version with the build profile and the web UI appended ("Ultimate.150-slim",
// "Ultimate.150-full-newui"); the default profile with the classic UI has no suffix.
// It is what --version and the start log show.
func Full() string {
	v := Version
	if Profile != "" {
		v += "-" + Profile
	}
	if UI != "" {
		v += "-" + UI
	}
	return v
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
