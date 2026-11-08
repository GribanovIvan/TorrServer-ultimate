//go:build slim

package torrstor

import (
	"errors"
	"sync"
)

// The slim profile keeps pieces in RAM only: diskpiece.go, diskhandles*.go and
// verified.go are not compiled in. useDisk is constant false, so the disk
// branches of piece.go and cache.go fold away; the DiskPiece below only keeps
// them compiling and is never created.

func useDisk() bool { return false }

var errNoDiskCache = errors.New("disk cache is not in this build")

type DiskPiece struct {
	name       string
	unverified bool
	mu         sync.RWMutex
}

func NewDiskPiece(*Piece) *DiskPiece                  { return nil }
func (*DiskPiece) WriteAt([]byte, int64) (int, error) { return 0, errNoDiskCache }
func (*DiskPiece) ReadAt([]byte, int64) (int, error)  { return 0, errNoDiskCache }
func (*DiskPiece) Close() error                       { return nil }
func (*DiskPiece) releaseState() string               { return "" }
func (*DiskPiece) needsCheck() bool                   { return false }
func (*DiskPiece) Release()                           {}
func closeAndRemove(string)                           {}
func verifiedPath(*Cache) string                      { return "" }
func (*Cache) loadVerified()                          {}
func (*Cache) markVerified(int, bool)                 {}
func (*Cache) flushVerified()                         {}
