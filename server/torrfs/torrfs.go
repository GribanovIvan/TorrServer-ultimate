//go:build !slim

package torrfs

func New() *RootDir {
	r := NewRootDir()
	return r
}
