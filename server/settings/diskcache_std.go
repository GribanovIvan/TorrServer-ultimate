//go:build !slim

package settings

// DiskCache is false in the slim profile, which keeps pieces in RAM only and
// ignores UseDisk.
const DiskCache = true
