package torrstor

import (
	"io"
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
		p.Complete = ff.Size() == p.cache.pieceLength
		p.Accessed = ff.ModTime().Unix()
	}
	return &DiskPiece{piece: p, name: name}
}

func (p *DiskPiece) WriteAt(b []byte, off int64) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ff, err := p.openLocked(true)
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	n, err = ff.WriteAt(b, off)

	p.piece.Size += int64(n)
	if p.piece.Size > p.piece.cache.pieceLength {
		p.piece.Size = p.piece.cache.pieceLength
	}
	p.piece.Accessed = time.Now().Unix()
	return
}

func (p *DiskPiece) ReadAt(b []byte, off int64) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ff, err := p.openLocked(false)
	if os.IsNotExist(err) {
		return 0, io.EOF
	}
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	n, err = ff.ReadAt(b, off)

	p.piece.Accessed = time.Now().Unix()
	if int64(len(b))+off >= p.piece.Size {
		go p.piece.cache.cleanPieces()
	}
	return n, err
}

// Keep handles across small torrent chunks; callers serialize access with mu.
func (p *DiskPiece) openLocked(write bool) (*os.File, error) {
	if p.piece.cache.isClosed.Load() {
		return nil, io.ErrClosedPipe
	}
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
	return file, nil
}

func (p *DiskPiece) closeLocked() error {
	if p.file == nil {
		return nil
	}
	file := p.file
	p.file, p.writable = nil, false
	return file.Close()
}

// Close retains downloaded bytes; Release evicts them.
func (p *DiskPiece) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked()
}

func (p *DiskPiece) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.closeLocked(); err != nil {
		log.TLogln("Error closing piece:", err)
	}

	p.piece.Size = 0
	p.piece.Complete = false

	os.Remove(p.name)
}
