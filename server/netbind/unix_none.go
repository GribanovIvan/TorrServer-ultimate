//go:build !slim && !full

package netbind

import (
	"errors"
	"net"
)

// The default profile binds TCP only: every host is a TCP address.

func IsUnix(string) bool { return false }

func unixPath(host string) string { return host }

func listenUnix(string) (net.Listener, error) {
	return nil, errors.New("unix sockets are not in this build")
}
