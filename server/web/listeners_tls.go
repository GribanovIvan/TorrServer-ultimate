//go:build !slim

package web

import (
	"bufio"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"server/log"
)

const (
	// sniffTimeout is how long a new connection on the HTTPS port may stay silent.
	sniffTimeout = 10 * time.Second
	// tlsRecordHandshake is the first byte of every TLS ClientHello.
	tlsRecordHandshake = 0x16
)

// splitTLS splits connections accepted on inner by their first byte: TLS handshakes go
// to tlsLn, anything else (plain HTTP sent to the HTTPS port) goes to plainLn.
// inner is closed once both returned listeners are closed.
func splitTLS(inner net.Listener) (tlsLn, plainLn net.Listener) {
	var open atomic.Int32
	open.Store(2)
	onClose := func() {
		if open.Add(-1) == 0 {
			_ = inner.Close()
		}
	}
	t := newConnQueue(inner.Addr(), onClose)
	p := newConnQueue(inner.Addr(), onClose)
	go func() {
		for {
			c, err := inner.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.TLogln("https accept error:", err)
				time.Sleep(50 * time.Millisecond) // e.g. out of file descriptors
				continue
			}
			// sniff in a goroutine so a silent client can't stall the accept loop
			go routeConn(c, t, p)
		}
	}()
	return t, p
}

func routeConn(c net.Conn, tlsQ, plainQ *connQueue) {
	br := bufio.NewReader(c)
	_ = c.SetReadDeadline(time.Now().Add(sniffTimeout))
	first, err := br.Peek(1)
	_ = c.SetReadDeadline(time.Time{})
	if err != nil {
		_ = c.Close()
		return
	}
	pc := &peekedConn{Conn: c, r: br}
	if first[0] == tlsRecordHandshake {
		tlsQ.push(pc)
	} else {
		plainQ.push(pc)
	}
}

// peekedConn replays bytes already buffered while sniffing.
type peekedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *peekedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// connQueue is a net.Listener fed by splitTLS.
type connQueue struct {
	conns   chan net.Conn
	done    chan struct{}
	once    sync.Once
	addr    net.Addr
	onClose func()
}

func newConnQueue(addr net.Addr, onClose func()) *connQueue {
	return &connQueue{conns: make(chan net.Conn), done: make(chan struct{}), addr: addr, onClose: onClose}
}

func (q *connQueue) push(c net.Conn) {
	select {
	case q.conns <- c:
	case <-q.done:
		_ = c.Close()
	}
}

func (q *connQueue) Accept() (net.Conn, error) {
	select {
	case c := <-q.conns:
		return c, nil
	case <-q.done:
		return nil, net.ErrClosed
	}
}

func (q *connQueue) Close() error {
	q.once.Do(func() {
		close(q.done)
		q.onClose()
	})
	return nil
}

func (q *connQueue) Addr() net.Addr { return q.addr }
