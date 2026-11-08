//go:build !slim

package torrstor

import "server/settings"

// useDisk reports whether pieces go to the disk cache (diskpiece.go) instead of
// RAM. The slim profile has no disk cache: see disk_slim.go.
func useDisk() bool { return settings.BTsets.UseDisk }
