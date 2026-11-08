package torrstor

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"server/log"
	"server/settings"
)

// verifiedFile records, per torrent cache directory, the pieces that passed the
// hash check, so they are trusted after a restart without re-reading them. A
// full-size file without that mark is re-checked by the torrent client
// (Completion.Ok=false): chunks arrive out of order, so a file of the right
// size can still have holes. Idea and bitmap format from pxlvoid/TorrServer.
const verifiedFile = ".verified.json"

type verifiedMeta struct {
	PieceLength int64  `json:"pieceLength"`
	PieceCount  int    `json:"pieceCount"`
	Verified    string `json:"verified"` // base64 bitmap, bit i = piece i
}

func encodeVerified(bits []bool) string {
	buf := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b {
			buf[i/8] |= 1 << (i % 8)
		}
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func decodeVerified(s string, n int) []bool {
	bits := make([]bool, n)
	buf, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return bits
	}
	for i := 0; i < n && i/8 < len(buf); i++ {
		bits[i] = buf[i/8]&(1<<(i%8)) != 0
	}
	return bits
}

func verifiedPath(c *Cache) string {
	return filepath.Join(settings.BTsets.TorrentsSavePath, c.hash.HexString(), verifiedFile)
}

// loadVerified reads the sidecar and applies it: pieces already complete by size
// but not verified lose Complete and ask for a hash check. Called from Cache.Init
// before the cache is handed to the client.
func (c *Cache) loadVerified() {
	verified := make([]bool, c.pieceCount)
	if buf, err := os.ReadFile(verifiedPath(c)); err == nil {
		var m verifiedMeta
		if json.Unmarshal(buf, &m) == nil && m.PieceLength == c.pieceLength && m.PieceCount == c.pieceCount {
			verified = decodeVerified(m.Verified, c.pieceCount)
		}
	}
	recheck := 0
	for i := 0; i < c.pieceCount; i++ {
		p := c.pieces[i]
		if p == nil || p.dPiece == nil {
			continue
		}
		if p.Complete && !verified[i] {
			p.Complete = false
			p.dPiece.unverified = true
			recheck++
		}
		verified[i] = verified[i] && p.Complete
	}
	c.muVerified.Lock()
	c.verified = verified
	c.muVerified.Unlock()
	if recheck > 0 {
		log.TLogln("Disk cache:", recheck, "pieces will be hash-checked:", c.hash.HexString())
	}
}

// markVerified updates the bitmap and schedules a debounced write.
func (c *Cache) markVerified(id int, ok bool) {
	c.muVerified.Lock()
	if id < 0 || id >= len(c.verified) || c.verified[id] == ok {
		c.muVerified.Unlock()
		return
	}
	c.verified[id] = ok
	schedule := !c.verifiedPending
	if schedule {
		c.verifiedPending = true
	}
	c.muVerified.Unlock()
	if schedule {
		time.AfterFunc(verifiedFlushDelay, c.flushVerifiedScheduled)
	}
}

const verifiedFlushDelay = 2 * time.Second

func (c *Cache) flushVerifiedScheduled() {
	c.muVerified.Lock()
	c.verifiedPending = false
	c.muVerified.Unlock()
	if !c.isClosed.Load() {
		c.flushVerified()
	}
}

// flushVerified writes the sidecar atomically. Holds no cache lock during I/O.
func (c *Cache) flushVerified() {
	c.muVerified.Lock()
	if c.verified == nil || c.pieceCount == 0 {
		c.muVerified.Unlock()
		return
	}
	bits := append([]bool(nil), c.verified...)
	c.muVerified.Unlock()

	buf, err := json.Marshal(verifiedMeta{PieceLength: c.pieceLength, PieceCount: c.pieceCount, Verified: encodeVerified(bits)})
	if err != nil {
		return
	}
	path := verifiedPath(c)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o666); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.TLogln("Error save verified pieces:", err)
	}
}
