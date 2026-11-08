//go:build !unix

package torrstor

func maxDiskHandles() int { return 4096 }
