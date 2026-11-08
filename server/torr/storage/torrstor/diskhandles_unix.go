//go:build unix

package torrstor

import "syscall"

const (
	minDiskHandles   = 64
	maxDiskHandleCap = 4096
)

// maxDiskHandles leaves most descriptors to peers, HTTP clients and logs. The Go
// runtime has already raised the soft limit to the hard one by this point.
func maxDiskHandles() int {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		return maxDiskHandleCap
	}
	n := lim.Cur / 4
	if n > maxDiskHandleCap {
		return maxDiskHandleCap
	}
	if n < minDiskHandles {
		return minDiskHandles
	}
	return int(n)
}
