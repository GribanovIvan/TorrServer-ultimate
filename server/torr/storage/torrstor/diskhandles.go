package torrstor

import (
	"container/list"
	"sync"
)

// diskHandleTable caps file handles retained by DiskPiece. A retained handle
// avoids open/close on each small chunk, but a large cache with small pieces
// could otherwise hold one descriptor per live piece and reach RLIMIT_NOFILE.
//
// Oldest-opened handles are closed first. Closing only costs a later reopen.
// Lock order is piece mu -> table mu; victims are closed with TryLock outside
// the table lock, so two pieces evicting each other cannot deadlock.
type diskHandleTable struct {
	mu    sync.Mutex
	max   int
	order *list.List
	open  map[*DiskPiece]*list.Element
}

func newDiskHandleTable(limit int) *diskHandleTable {
	return &diskHandleTable{max: limit, order: list.New(), open: make(map[*DiskPiece]*list.Element)}
}

var diskHandles = newDiskHandleTable(maxDiskHandles())

// add records p as holding a handle and returns the oldest entries over the
// limit. They stay listed until actually closed, so a busy victim is retried.
func (t *diskHandleTable) add(p *DiskPiece) []*DiskPiece {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.open[p]; !ok {
		t.open[p] = t.order.PushBack(p)
	}
	over := t.order.Len() - t.max
	if over <= 0 {
		return nil
	}
	victims := make([]*DiskPiece, 0, over)
	for e := t.order.Front(); e != nil && len(victims) < over; e = e.Next() {
		if v := e.Value.(*DiskPiece); v != p {
			victims = append(victims, v)
		}
	}
	return victims
}

func (t *diskHandleTable) remove(p *DiskPiece) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if e, ok := t.open[p]; ok {
		t.order.Remove(e)
		delete(t.open, p)
	}
}

// evictIdle closes the victim's handle unless the piece is busy.
func (p *DiskPiece) evictIdle() {
	if !p.mu.TryLock() {
		return
	}
	defer p.mu.Unlock()
	_ = p.closeLocked()
}
