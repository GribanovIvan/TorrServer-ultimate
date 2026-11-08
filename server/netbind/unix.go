//go:build slim || full

package netbind

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Unix-domain sockets are in the slim and full profiles; the default one binds
// TCP only (unix_none.go).

// unixPrefix marks a bind host as a Unix-domain socket path: unix:/run/ts.sock
// (--ip unix:/run/ts.sock). Other hosts are TCP.
const unixPrefix = "unix:"

// IsUnix reports whether host names a Unix-domain socket.
func IsUnix(host string) bool {
	return strings.HasPrefix(host, unixPrefix)
}

func unixPath(host string) string {
	return strings.TrimPrefix(host, unixPrefix)
}

// listenUnix binds a socket path, replacing a stale socket left by a crash.
// A path with a live listener, or a non-socket file, is an error.
func listenUnix(path string) (net.Listener, error) {
	if path == "" {
		return nil, fmt.Errorf("unix socket: empty path")
	}
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("%s exists and is not a socket", path)
		}
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			_ = conn.Close()
			return nil, fmt.Errorf("%s: already in use", path)
		}
		_ = os.Remove(path)
	}
	return net.Listen("unix", path)
}
