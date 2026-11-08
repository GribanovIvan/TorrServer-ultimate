//go:build !slim

package offlinetest

import (
	"path/filepath"

	"server/ffprobe"
)

// hideFFprobe points ffprobe at a missing binary so tests do not depend on one
// being installed. It returns the restore function.
func hideFFprobe(dir string) func() {
	return ffprobe.SetBinPath(filepath.Join(dir, "no-ffprobe"))
}
