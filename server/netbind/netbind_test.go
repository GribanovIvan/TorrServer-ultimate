package netbind

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestNormalizeEmpty(t *testing.T) {
	got := Normalize(nil)
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("Normalize(nil) = %#v, want [\"\"]", got)
	}
}

func TestNormalizeDedup(t *testing.T) {
	got := Normalize([]string{"127.0.0.1", "127.0.0.1", "::1"})
	if len(got) != 2 {
		t.Fatalf("Normalize dedup = %#v, want 2 entries", got)
	}
}

func TestAddr(t *testing.T) {
	if got := Addr("", "8090"); got != ":8090" {
		t.Fatalf("Addr empty host = %q", got)
	}
	if got := Addr("127.0.0.1", "8090"); got != "127.0.0.1:8090" {
		t.Fatalf("Addr ipv4 = %q", got)
	}
	if got := Addr("::1", "8090"); got != "[::1]:8090" {
		t.Fatalf("Addr ipv6 = %q", got)
	}
}

func TestListenMultiple(t *testing.T) {
	l, err := Listen([]string{"127.0.0.1"}, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	port := l.Addr().(*net.TCPAddr).Port
	conn, err := net.Dial("tcp", Addr("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
}

func TestMultiListenerAcceptAndClose(t *testing.T) {
	a, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ml := &multiListener{
		listeners: []net.Listener{a, b},
		conns:     make(chan net.Conn),
		errs:      make(chan error, 2),
	}
	go ml.acceptLoop(a)
	go ml.acceptLoop(b)
	t.Cleanup(func() { _ = ml.Close() })

	if ml.Addr() == nil {
		t.Fatal("addr")
	}
	for _, ln := range ml.listeners {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		got, err := ml.Accept()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(c, "x"); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 1)
		if _, err := got.Read(buf); err != nil || buf[0] != 'x' {
			t.Fatalf("read %q %v", buf, err)
		}
		_ = c.Close()
		_ = got.Close()
	}
	if err := ml.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ml.Close(); err != nil {
		t.Fatal("second close")
	}
	if _, err := ml.Accept(); err == nil {
		t.Fatal("accept after close")
	}
}

func TestCheckPortBusy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	if err := CheckPort([]string{"127.0.0.1"}, port); err == nil {
		t.Fatal("expected busy port")
	}
}

func TestListenTwoHosts(t *testing.T) {
	ln, err := Listen([]string{"127.0.0.1", "127.0.0.2"}, "0")
	if err != nil {
		t.Skip("dual loopback bind unavailable:", err)
	}
	defer func() { _ = ln.Close() }()
	ml := ln.(*multiListener)
	if len(ml.listeners) != 2 || ml.Addr() == nil {
		t.Fatalf("listeners %d", len(ml.listeners))
	}
}

func TestMultiListenerConcurrent(t *testing.T) {
	a, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ml := &multiListener{listeners: []net.Listener{a}, conns: make(chan net.Conn), errs: make(chan error, 1)}
	go ml.acceptLoop(a)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		c, err := ml.Accept()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			errCh <- err
			return
		}
		if c != nil {
			_ = c.Close()
		}
	}()
	c, err := net.Dial("tcp", a.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
	_ = ml.Close()
	empty := &multiListener{}
	if empty.Addr() != nil {
		t.Fatal("empty addr")
	}
}

func TestListenUnixServesAndUnlinksOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ts.sock")
	l, err := Listen([]string{unixPrefix + path}, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		_, _ = c.Write([]byte("ok"))
		_ = c.Close()
	}()
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	buf := make([]byte, 2)
	if _, err := io.ReadFull(c, buf); err != nil || string(buf) != "ok" {
		t.Fatalf("read = %q, %v", buf, err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket left after Close: %v", err)
	}
}

func TestListenUnixReplacesStaleSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ts.sock")
	stale, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a crash: stop listening but leave the socket file behind.
	stale.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = stale.Close()
	if err := CheckPort([]string{unixPrefix + path}, ""); err != nil {
		t.Fatalf("stale socket not replaced: %v", err)
	}
}

func TestListenUnixLiveOrPlainFileIsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ts.sock")
	live, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = live.Close() }()
	if err := CheckPort([]string{unixPrefix + path}, ""); err == nil {
		t.Fatal("expected error for a socket with a live listener")
	}
	plain := filepath.Join(dir, "file")
	if err := os.WriteFile(plain, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckPort([]string{unixPrefix + plain}, ""); err == nil {
		t.Fatal("expected error for a non-socket path")
	}
	if _, err := os.Stat(plain); err != nil {
		t.Fatalf("plain file must be left alone: %v", err)
	}
}

func TestUnixHostHelpers(t *testing.T) {
	if !HasTCP(nil) || !HasTCP([]string{"127.0.0.1", "unix:/x"}) {
		t.Fatal("HasTCP should be true")
	}
	if HasTCP([]string{"unix:/x"}) {
		t.Fatal("HasTCP false for unix-only")
	}
	if got := TCPHosts([]string{"127.0.0.1", "unix:/x"}); len(got) != 1 || got[0] != "127.0.0.1" {
		t.Fatalf("TCPHosts = %#v", got)
	}
	if got := Addr("unix:/x", "8090"); got != "unix:/x" {
		t.Fatalf("Addr unix = %q", got)
	}
}
