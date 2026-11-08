package netbind

import (
	"fmt"
	"net"
	"sync"
)

// HasTCP reports whether binding hosts opens at least one TCP listener.
// An empty list binds all interfaces, so it counts.
func HasTCP(hosts []string) bool {
	if len(hosts) == 0 {
		return true
	}
	for _, h := range hosts {
		if !IsUnix(h) {
			return true
		}
	}
	return false
}

// TCPHosts returns hosts without the Unix-socket entries.
func TCPHosts(hosts []string) []string {
	out := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if !IsUnix(h) {
			out = append(out, h)
		}
	}
	return out
}

func listenHost(host, port string) (net.Listener, error) {
	if IsUnix(host) {
		return listenUnix(unixPath(host))
	}
	return net.Listen("tcp", Addr(host, port))
}

// Normalize returns unique bind hosts. An empty list means all interfaces once.
func Normalize(ips []string) []string {
	if len(ips) == 0 {
		return []string{""}
	}
	out := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	return out
}

// Addr formats host:port for net.Listen.
func Addr(host, port string) string {
	if IsUnix(host) {
		return host
	}
	if host == "" {
		return ":" + port
	}
	return net.JoinHostPort(host, port)
}

// CheckPort verifies that TCP port is available on every bind host; a Unix path
// is checked for being free the same way.
func CheckPort(hosts []string, port string) error {
	for _, host := range Normalize(hosts) {
		addr := Addr(host, port)
		l, err := listenHost(host, port)
		if err != nil {
			return fmt.Errorf("%s: %w", addr, err)
		}
		_ = l.Close()
	}
	return nil
}

// Listen binds TCP (or, in the slim and full profiles, a Unix socket: unix.go) on each host and returns a listener that accepts from all of them.
func Listen(hosts []string, port string) (net.Listener, error) {
	hosts = Normalize(hosts)
	if len(hosts) == 1 {
		return listenHost(hosts[0], port)
	}

	ml := &multiListener{
		conns: make(chan net.Conn),
		errs:  make(chan error, len(hosts)),
	}
	for _, host := range hosts {
		l, err := listenHost(host, port)
		if err != nil {
			_ = ml.Close()
			return nil, err
		}
		ml.listeners = append(ml.listeners, l)
		go ml.acceptLoop(l)
	}
	return ml, nil
}

type multiListener struct {
	listeners []net.Listener
	conns     chan net.Conn
	errs      chan error
	closeOnce sync.Once
}

func (ml *multiListener) acceptLoop(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			select {
			case ml.errs <- err:
			default:
			}
			return
		}
		ml.conns <- c
	}
}

func (ml *multiListener) Accept() (net.Conn, error) {
	select {
	case c := <-ml.conns:
		return c, nil
	case err := <-ml.errs:
		return nil, err
	}
}

func (ml *multiListener) Close() error {
	var err error
	ml.closeOnce.Do(func() {
		for _, l := range ml.listeners {
			if closeErr := l.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
	})
	return err
}

func (ml *multiListener) Addr() net.Addr {
	if len(ml.listeners) == 0 {
		return nil
	}
	return ml.listeners[0].Addr()
}
