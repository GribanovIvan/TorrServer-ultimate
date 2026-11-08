//go:build slim || full

package netbind

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

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
