package torrstor

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anacrolix/torrent"

	"server/log"
	"server/settings"
	"server/torr/storage/state"
	"server/torr/utils"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

type Cache struct {
	storage.TorrentImpl
	storage *Storage

	capacity int64
	filled   atomic.Int64
	hash     metainfo.Hash

	pieceLength int64
	pieceCount  int

	// pieces content is immutable after Init; muPieces guards the map
	// reference itself (nilled in Close), so holders of a snapshot may
	// safely iterate it without the lock
	pieces   map[int]*Piece
	muPieces sync.RWMutex

	readers   map[*Reader]struct{}
	muReaders sync.RWMutex

	isRemove atomic.Bool
	isClosed atomic.Bool
	muRemove sync.Mutex
	// muPrio serializes clearPriority and setLoadPriority so that the priority
	// reset of a reader that has just closed cannot wipe the priorities a
	// freshly created reader has already set.
	muPrio sync.Mutex
	// prioritized tracks piece ids this cache has set a priority on (guarded by
	// muPrio). Every SetPriority/PieceState call takes the anacrolix client
	// lock - the same lock all peer traffic serializes on - so priority resets
	// must touch only these pieces, never the whole torrent.
	prioritized map[int]struct{}
	lastClean   atomic.Int64
	lastGC      atomic.Int64
	buffPool    sync.Pool
	torrent     *torrent.Torrent
}

func NewCache(capacity int64, storage *Storage) *Cache {
	ret := &Cache{
		capacity:    capacity,
		pieces:      make(map[int]*Piece),
		storage:     storage,
		readers:     make(map[*Reader]struct{}),
		prioritized: make(map[int]struct{}),
	}

	return ret
}

func (c *Cache) Init(info *metainfo.Info, hash metainfo.Hash) {
	log.TLogln("Create cache for:", info.Name, hash.HexString())
	if c.capacity == 0 {
		c.capacity = info.PieceLength * 4
	}

	c.pieceLength = info.PieceLength
	c.pieceCount = info.NumPieces()
	c.hash = hash

	if settings.BTsets.UseDisk {
		name := filepath.Join(settings.BTsets.TorrentsSavePath, hash.HexString())
		err := os.MkdirAll(name, 0o777)
		if err != nil {
			log.TLogln("Error create dir:", err)
		}
	}

	for i := 0; i < c.pieceCount; i++ {
		c.pieces[i] = NewPiece(i, c)
	}

	go c.priorityWatchdog()
}

// priorityWatchdog re-arms piece priorities while readers are active.
//
// setLoadPriority is only reached through the cache cleanup path, which is
// driven by piece reads and writes (see mempiece.go and diskpiece.go). Should
// priorities ever end up cleared while a reader still needs data, nothing is
// downloaded, so no piece I/O happens, so cleanup never runs and the
// priorities are never restored - the torrent stalls indefinitely with peers
// connected. Re-arming them periodically breaks that cycle regardless of how
// the priorities were lost.
func (c *Cache) priorityWatchdog() {
	for {
		time.Sleep(5 * time.Second)
		if c.isClosed.Load() {
			return
		}
		if c.torrent == nil {
			continue
		}
		if c.GetUseReaders() > 0 {
			c.getRemPieces()
		}
	}
}

func (c *Cache) SetTorrent(torr *torrent.Torrent) {
	c.torrent = torr
}

// getPieceBuffer reuses released piece buffers to avoid multi-MB allocations
// on every piece, which causes GC stalls on low-RAM devices. Buffers are
// zeroed on reuse so responsive-mode reads of partially written pieces see
// the same zero bytes a fresh allocation would give.
func (c *Cache) getPieceBuffer() []byte {
	if buf, ok := c.buffPool.Get().([]byte); ok && int64(len(buf)) == c.pieceLength {
		clear(buf)
		return buf
	}
	return make([]byte, c.pieceLength)
}

func (c *Cache) putPieceBuffer(buf []byte) {
	if int64(len(buf)) == c.pieceLength {
		c.buffPool.Put(buf)
	}
}

func (c *Cache) getPieces() map[int]*Piece {
	c.muPieces.RLock()
	defer c.muPieces.RUnlock()
	return c.pieces
}

func (c *Cache) readersSnapshot() []*Reader {
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	list := make([]*Reader, 0, len(c.readers))
	for r := range c.readers {
		list = append(list, r)
	}
	return list
}

func (c *Cache) Piece(m metainfo.Piece) storage.PieceImpl {
	if val, ok := c.getPieces()[m.Index()]; ok {
		return val
	}
	return &PieceFake{}
}

func (c *Cache) Close() error {
	if c.torrent != nil {
		log.TLogln("Close cache for:", c.torrent.Name(), c.hash)
	} else {
		log.TLogln("Close cache for:", c.hash)
	}
	c.isClosed.Store(true)

	c.storage.removeCache(c.hash)

	if settings.BTsets.RemoveCacheOnDrop {
		name := filepath.Join(settings.BTsets.TorrentsSavePath, c.hash.HexString())
		if name != "" && name != "/" {
			for _, v := range c.getPieces() {
				if v.dPiece != nil {
					os.Remove(v.dPiece.name)
				}
			}
			os.Remove(name)
		}
	}

	c.muReaders.Lock()
	c.readers = nil
	c.muReaders.Unlock()

	c.muPieces.Lock()
	c.pieces = nil
	c.muPieces.Unlock()

	utils.FreeOSMemGC()
	return nil
}

func (c *Cache) removePiece(piece *Piece) {
	if !c.isClosed.Load() {
		piece.Release()
	}
}

func (c *Cache) AdjustRA(readahead int64) {
	if c == nil {
		return
	}
	if settings.BTsets.CacheSize == 0 {
		c.capacity = readahead * 3
	}
	if readahead > c.capacity {
		readahead = c.capacity
	}
	// Called every second per torrent; SetReadahead takes the anacrolix
	// reader lock, so skip readers already at the requested value.
	for _, r := range c.readersSnapshot() {
		if r.Readahead() != readahead {
			r.SetReadahead(readahead)
		}
	}
}

func (c *Cache) GetState() *state.CacheState {
	cState := new(state.CacheState)

	piecesState := make(map[int]state.ItemState, 0)
	var fill int64 = 0

	for _, p := range c.getPieces() {
		if p.Size > 0 {
			fill += p.Size
			piecesState[p.Id] = state.ItemState{
				Id:        p.Id,
				Size:      p.Size,
				Length:    c.pieceLength,
				Completed: p.Complete,
				Priority:  int(c.torrent.PieceState(p.Id).Priority),
			}
		}
	}

	readersState := make([]*state.ReaderState, 0)

	for _, r := range c.readersSnapshot() {
		rng := r.getPiecesRange()
		pc := r.getReaderPiece()
		readersState = append(readersState, &state.ReaderState{
			Start:  rng.Start,
			End:    rng.End,
			Reader: pc,
		})
	}

	c.filled.Store(fill)
	cState.Capacity = c.capacity
	cState.PiecesLength = c.pieceLength
	cState.PiecesCount = c.pieceCount
	cState.Hash = c.hash.HexString()
	cState.Filled = fill
	cState.Pieces = piecesState
	cState.Readers = readersState
	return cState
}

func (c *Cache) cleanPieces() {
	if c.isRemove.Load() || c.isClosed.Load() {
		return
	}

	// Piece reads and writes trigger this many times per second; a full pass
	// is only needed immediately when the cache is over capacity, otherwise
	// rate-limit it to keep client-lock pressure off the download path.
	now := time.Now().UnixMilli()
	if now-c.lastClean.Load() < 500 && c.filled.Load() <= c.capacity {
		return
	}

	// Protection against concurrent deletion
	if !c.muRemove.TryLock() {
		return // Cleanup is already in progress in another goroutine
	}
	defer c.muRemove.Unlock()

	c.isRemove.Store(true)
	defer func() { c.isRemove.Store(false) }()
	c.lastClean.Store(now)

	remPieces := c.getRemPieces()
	filled := c.filled.Load()
	if filled > c.capacity {
		rems := (filled-c.capacity)/c.pieceLength + 1
		for _, p := range remPieces {
			c.removePiece(p)
			rems--
			if rems <= 0 {
				c.freeMemRateLimited()
				return
			}
		}
	}
}

// Forced GC is two stop-the-world passes and takes hundreds of milliseconds
// on weak ARM CPUs; evictions run up to twice a second while streaming, so
// return memory to the OS at most every 30 seconds. In between, the Go
// background scavenger still releases free pages gradually.
func (c *Cache) freeMemRateLimited() {
	now := time.Now().Unix()
	last := c.lastGC.Load()
	if now-last >= 30 && c.lastGC.CompareAndSwap(last, now) {
		utils.FreeOSMemGC()
	}
}

func (c *Cache) getRemPieces() []*Piece {
	readers := c.readersSnapshot()

	// Active readers drive downloads; idle readers (e.g. a paused player that
	// stopped reading the stream) still keep their offset, so their windows
	// must stay cached or resume starts from a cold cache and stalls.
	ranges := make([]Range, 0)
	keepRanges := make([]Range, 0)
	for _, r := range readers {
		r.checkReader()
		if r.isUse {
			rng := r.getPiecesRange()
			ranges = append(ranges, rng)
			keepRanges = append(keepRanges, rng)
		} else {
			// An idle reader keeps only a small resume window; its full
			// cache-sized range would let every stale player connection pin
			// CacheSize bytes in RAM, blowing past the configured limit.
			keepRanges = append(keepRanges, r.getResumeRange())
		}
	}
	ranges = mergeRange(ranges)
	keepRanges = mergeRange(keepRanges)

	piecesRemove := make([]*Piece, 0)
	fill := int64(0)

	// Determine which chunks can be deleted
	for id, p := range c.getPieces() {
		if p.Size > 0 {
			fill += p.Size
		}
		if len(keepRanges) > 0 {
			if !inRanges(keepRanges, id) {
				if p.Size > 0 && !c.isIdInFileBE(keepRanges, id) {
					piecesRemove = append(piecesRemove, p)
				}
			}
		} else {
			// When preloading, clear everything except the beginning and end of the file
			if p.Size > 0 && !c.isIdInFileBE(keepRanges, id) {
				piecesRemove = append(piecesRemove, p)
			}
		}
	}

	c.clearPriority()
	c.setLoadPriority(ranges)

	// Sort by last access time (oldest first)
	sort.Slice(piecesRemove, func(i, j int) bool {
		return piecesRemove[i].Accessed < piecesRemove[j].Accessed
	})

	c.filled.Store(fill)
	return piecesRemove
}

func (c *Cache) setLoadPriority(ranges []Range) {
	readers := c.readersSnapshot()
	pieces := c.getPieces()
	if len(readers) == 0 || pieces == nil {
		return
	}
	c.muPrio.Lock()
	defer c.muPrio.Unlock()
	for _, r := range readers {
		if !r.isUse {
			continue
		}
		if c.isIdInFileBE(ranges, r.getReaderPiece()) {
			continue
		}
		readerPos := r.getReaderPiece()
		readerRAHPos := r.getReaderRAHPiece()
		end := r.getPiecesRange().End
		// Pipeline as many pieces as the reader's share of the cache window can
		// hold - the window itself bounds memory use, so a piece-count cap tied
		// to ConnectionsLimit only starves peers of outstanding requests.
		count := int(c.capacity/c.pieceLength)/len(readers) + 1
		limit := 0
		for i := readerPos; i < end && limit < count; i++ {
			if !pieces[i].Complete {
				if i == readerPos {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNow)
				} else if i == readerPos+1 {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNext)
				} else if i > readerPos && i <= readerRAHPos {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityReadahead)
				} else if i > readerRAHPos && i <= readerRAHPos+5 && c.torrent.PieceState(i).Priority != torrent.PiecePriorityHigh {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityHigh)
				} else if i > readerRAHPos+5 && c.torrent.PieceState(i).Priority != torrent.PiecePriorityNormal {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNormal)
				}
				c.prioritized[i] = struct{}{}
				limit++
			}
		}
	}
}

func (c *Cache) isIdInFileBE(ranges []Range, id int) bool {
	// keep 8/16 MB
	FileRangeNotDelete := int64(c.pieceLength)
	if FileRangeNotDelete < 8<<20 {
		FileRangeNotDelete = 8 << 20
	}

	for _, rng := range ranges {
		ss := int(rng.File.Offset() / c.pieceLength)
		se := int((rng.File.Offset() + FileRangeNotDelete) / c.pieceLength)

		es := int((rng.File.Offset() + rng.File.Length() - FileRangeNotDelete) / c.pieceLength)
		ee := int((rng.File.Offset() + rng.File.Length()) / c.pieceLength)

		if id >= ss && id < se || id > es && id <= ee {
			return true
		}
	}
	return false
}

//////////////////
// Reader section
////////

func (c *Cache) NewReader(file *torrent.File) *Reader {
	return newReader(file, c)
}

func (c *Cache) GetUseReaders() int {
	if c == nil {
		return 0
	}
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	readers := 0
	for reader := range c.readers {
		if reader.isUse {
			readers++
		}
	}
	return readers
}

func (c *Cache) Readers() int {
	if c == nil {
		return 0
	}
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	return len(c.readers)
}

func (c *Cache) CloseReader(r *Reader) {
	r.cache.muReaders.Lock()
	delete(r.cache.readers, r)
	r.cache.muReaders.Unlock()
	// Reader.Close touches anacrolix internals, keep it outside muReaders
	r.Close()
	go c.clearPriority()
}

func (c *Cache) clearPriority() {
	if c.torrent == nil {
		return
	}
	// This used to sleep for a second before clearing priorities. A reader
	// created during that window could have its PiecePriorityNow/Next/Readahead
	// reset to None right after setLoadPriority had assigned them, starving the
	// player. A mutex provides the same ordering without the race window.
	c.muPrio.Lock()
	defer c.muPrio.Unlock()
	ranges := make([]Range, 0)
	for _, r := range c.readersSnapshot() {
		r.checkReader()
		if r.isUse {
			ranges = append(ranges, r.getPiecesRange())
		}
	}
	ranges = mergeRange(ranges)

	// Only pieces this cache prioritized need resetting; anything else is
	// already at PiecePriorityNone, and touching it would take the client
	// lock once per piece across the whole torrent.
	for id := range c.prioritized {
		if !inRanges(ranges, id) {
			c.torrent.Piece(id).SetPriority(torrent.PiecePriorityNone)
			delete(c.prioritized, id)
		}
	}
}

func (c *Cache) GetCapacity() int64 {
	if c == nil {
		return 0
	}
	return c.capacity
}
