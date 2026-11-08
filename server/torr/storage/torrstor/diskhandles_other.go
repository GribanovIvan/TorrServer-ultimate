//go:build !unix && !slim

package torrstor

func maxDiskHandles() int { return 4096 }
