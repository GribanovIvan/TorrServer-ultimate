package torrstor

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"server/log"
	"server/settings"
)

type DiskPiece struct {
	piece *Piece

	name     string
	file     *os.File
	writable bool

	mu sync.RWMutex
}

func NewDiskPiece(p *Piece) *DiskPiece {
	name := filepath.Join(settings.BTsets.TorrentsSavePath, p.cache.hash.HexString(), strconv.Itoa(p.Id))
	ff, err := os.Stat(name)
	if err == nil {
		p.Size = ff.Size()
		plen := p.cache.pieceByteLength(p.Id)
		p.Complete = plen > 0 && ff.Size() >= plen
		p.Accessed = ff.ModTime().Unix()
		if p.Size > 0 {
			p.cache.notePieceFilled(p.Id)
		}
	}
	return &DiskPiece{piece: p, name: name}
}

func (p *DiskPiece) WriteAt(b []byte, off int64) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Same as MemPiece: isClosed under piece mu vs Close→Release.
	if p.piece.cache.isClosed.Load() {
		return 0, os.ErrNotExist
	}
	ff, err := p.openLocked(true)
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	n, err = ff.WriteAt(b, off)

	// Accumulate: chunks arrive out of order, so a high-water mark would count
	// gaps as filled. MarkNotComplete resets Size on retransmit.
	p.piece.Size += int64(n)
	plen := p.piece.cache.pieceByteLength(p.piece.Id)
	if plen > 0 && p.piece.Size > plen {
		p.piece.Size = plen
	}
	if p.piece.Size > 0 {
		p.piece.cache.notePieceFilled(p.piece.Id)
	}
	p.piece.Accessed = time.Now().Unix()
	return
}

func (p *DiskPiece) ReadAt(b []byte, off int64) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.piece.cache.isClosed.Load() {
		return 0, os.ErrNotExist
	}
	ff, err := p.openLocked(false)
	if os.IsNotExist(err) {
		return 0, err
	}
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	n, err = ff.ReadAt(b, off)

	p.piece.Accessed = time.Now().Unix()
	if p.piece.Complete && int64(len(b))+off >= p.piece.Size {
		go p.piece.cache.cleanPieces()
	}
	return n, err
}

// Keep handles across small torrent chunks; callers serialize access with mu.
func (p *DiskPiece) openLocked(write bool) (*os.File, error) {
	if p.file != nil && (!write || p.writable) {
		return p.file, nil
	}
	if err := p.closeLocked(); err != nil {
		return nil, err
	}
	flags := os.O_RDONLY
	if write {
		flags = os.O_RDWR | os.O_CREATE
	}
	file, err := os.OpenFile(p.name, flags, 0o666)
	if err != nil {
		return nil, err
	}
	p.file, p.writable = file, write
	for _, v := range diskHandles.add(p) {
		v.evictIdle()
	}
	return file, nil
}

func (p *DiskPiece) closeLocked() error {
	if p.file == nil {
		return nil
	}
	file := p.file
	p.file, p.writable = nil, false
	diskHandles.remove(p)
	return file.Close()
}

// Close retains downloaded bytes; Release evicts them.
func (p *DiskPiece) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked()
}

// detachedFiles holds handles taken off pieces by releaseState until
// closeAndRemove runs: releaseState can run under Cache.muRemove, and neither
// close(2) nor unlink may happen while that is held (AGENTS). Windows needs the
// handle closed before the file can be removed, hence close then unlink here.
var detachedFiles sync.Map // name string -> *os.File

// releaseState clears Size/Complete and returns the on-disk path.
// Callers that hold Cache.muRemove must call closeAndRemove only after unlocking.
func (p *DiskPiece) releaseState() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if f := p.file; f != nil {
		p.file, p.writable = nil, false
		diskHandles.remove(p)
		if old, loaded := detachedFiles.Swap(p.name, f); loaded {
			_ = old.(*os.File).Close()
		}
	}
	name := p.name
	p.piece.Size = 0
	p.piece.Complete = false
	p.piece.cache.notePieceEmpty(p.piece.Id)
	return name
}

// closeAndRemove closes a handle detached by releaseState, then unlinks the file.
func closeAndRemove(name string) {
	if f, ok := detachedFiles.LoadAndDelete(name); ok {
		_ = f.(*os.File).Close()
	}
	_ = os.Remove(name)
}

func (p *DiskPiece) Release() {
	name := p.releaseState()
	if name != "" {
		closeAndRemove(name)
	}
}
