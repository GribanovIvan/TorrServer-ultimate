// EN: Disable MPTCP to reduce CVE-2026-98432 RCE risk: bug is in IPv4 TCP, but as of 2026-11-05 RCE is confirmed with MPTCP (see link below)
// RU: Отключаем MPTCP для снижения риска RCE по CVE-2026-98432: ошибка в IPv4 TCP, но на 2026-11-05 RCE подтверждён при MPTCP (см. ссылку ниже)
// reference: https://www.anthropic.com/research/discovering-vulnerabilities-with-claude section "5. Zero-click remote code execution in Linux"
//go:debug multipathtcp=0

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"

	"server"
	"server/log"
	"server/settings"
	"server/version"
)

type args struct {
	Port string `arg:"-p" help:"web server port (default 8090)"`
	// bindArgs holds --ip (args_bind*.go): Unix sockets are in slim and full only.
	bindArgs
	Path        string `arg:"-d" help:"database and config dir path"`
	LogPath     string `arg:"-l" help:"server log file path"`
	WebLogPath  string `arg:"-w" help:"web access log file path"`
	RDB         bool   `arg:"-r" help:"start in read-only DB mode"`
	HttpAuth    bool   `arg:"-a" help:"enable http auth on all requests"`
	DontKill    bool   `arg:"-k" help:"don't kill server on signal"`
	TorrentAddr string `help:"Torrent client address, like 127.0.0.1:1337 (default :PeersListenPort)"`
	PubIPv4     string `arg:"-4" help:"set public IPv4 addr"`
	PubIPv6     string `arg:"-6" help:"set public IPv6 addr"`
	MaxSize     string `arg:"-m" help:"max allowed stream size (in Bytes)"`
	ProxyURL    string `help:"proxy URL for BitTorrent traffic (http, socks4, socks5, socks5h), e.g. socks5://user:password@127.0.0.1:8080"`
	ProxyMode   string `help:"proxy mode: tracker (only HTTP trackers, default), peers (only peer connections), or full (all traffic)"`
	// featureArgs holds the flags of the build profile (args_std.go, args_slim.go).
	featureArgs
}

func (args) Version() string {
	return "TorrServer " + version.Full()
}

var params args

// exitFunc terminates the process. Tests replace it.
var exitFunc = os.Exit

func main() {
	exitFunc(run(os.Args[1:]))
}

func run(argv []string) int {
	runtime.GOMAXPROCS(runtime.NumCPU())

	params = args{}
	p, err := arg.NewParser(arg.Config{}, &params)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err)
		return 2
	}
	err = p.Parse(argv)
	switch {
	case err == arg.ErrHelp:
		p.WriteHelp(os.Stdout)
		return 0
	case err == arg.ErrVersion:
		_, _ = fmt.Fprintln(os.Stdout, params.Version())
		return 0
	case err != nil:
		p.WriteUsage(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "error:", err)
		return 2
	}

	if params.Path == "" {
		params.Path, _ = os.Getwd()
	}

	if params.Port == "" {
		params.Port = settings.DefaultPort
	}

	settings.Path = params.Path
	settings.HttpAuth = params.HttpAuth
	log.Init(params.LogPath, params.WebLogPath)

	log.TLogln("=========== START ===========")
	log.TLogln("TorrServer", version.Full()+",", runtime.Version()+",", "CPU Num:", runtime.NumCPU())
	if msg := configureCARoots(); msg != "" {
		log.TLogln(msg)
	}
	if params.HttpAuth {
		log.TLogln("Use HTTP Auth file", settings.Path+"/accs.db")
	}
	if params.RDB {
		log.TLogln("Running in Read-only DB mode!")
	}
	// Simple Usage:
	dnsResolve()

	// Advanced Usage:
	// config := DNSConfig{
	//     PrimaryServers: []string{"1.1.1.1:53", "8.8.8.8:53"},
	//     Timeout:        3 * time.Second,
	// }
	// checker := NewDNSChecker(config)
	// // Perform DNS lookup with automatic fallback
	// addrs, err := checker.LookupHostWithFallback("themoviedb.org")
	// if err != nil {
	//     log.TLogln("DNS lookup failed:", err)
	// } else {
	// 	fmt.Println("DNS resolved:", addrs)
	// }

	Preconfig(params.DontKill)

	if params.TorrentAddr != "" {
		settings.TorAddr = params.TorrentAddr
	}

	if params.PubIPv4 != "" {
		settings.PubIPv4 = params.PubIPv4
	}

	if params.PubIPv6 != "" {
		settings.PubIPv6 = params.PubIPv6
	}

	if params.MaxSize != "" {
		maxSize, err := strconv.ParseInt(params.MaxSize, 10, 64)
		if err == nil {
			settings.MaxSize = maxSize
		}
	}

	if params.ProxyURL != "" && params.ProxyMode == "" {
		params.ProxyMode = "tracker" // default
	}
	if params.ProxyMode != "" && params.ProxyMode != "tracker" && params.ProxyMode != "peers" && params.ProxyMode != "full" {
		log.TLogln("Invalid proxy mode, using default 'tracker'")
		params.ProxyMode = "tracker"
	}

	settings.Args = &settings.ExecArgs{
		Port:        params.Port,
		IPs:         params.IPs,
		Path:        params.Path,
		LogPath:     params.LogPath,
		WebLogPath:  params.WebLogPath,
		RDB:         params.RDB,
		HttpAuth:    params.HttpAuth,
		DontKill:    params.DontKill,
		TorrentAddr: params.TorrentAddr,
		PubIPv4:     params.PubIPv4,
		PubIPv6:     params.PubIPv6,
		MaxSize:     params.MaxSize,
		ProxyURL:    params.ProxyURL,
		ProxyMode:   params.ProxyMode,
	}
	params.featureArgs.apply(settings.Args)

	if params.ProxyURL != "" {
		log.TLogln("Proxy configured from CLI:", params.ProxyURL, "mode:", settings.Args.ProxyMode)
	}

	if err := params.featureArgs.check(); err != nil {
		log.TLogln("Error:", err)
		return 1
	}
	params.featureArgs.start(params.Port)

	if err := server.Start(); err != nil {
		log.TLogln(err)
		return 1
	}
	log.TLogln(server.WaitServer())
	log.Close()
	time.Sleep(time.Second * 3)
	return 0
}

///////////
/// DNS
///

// DNSConfig holds DNS resolver configuration
type DNSConfig struct {
	PrimaryServers  []string
	FallbackServers []string
	Timeout         time.Duration
	CacheDuration   time.Duration
}

// DefaultDNSConfig returns a sensible default configuration
func DefaultDNSConfig() DNSConfig {
	return DNSConfig{
		PrimaryServers: []string{
			"9.9.9.9:53", // Quad9 DNS
			"8.8.8.8:53", // Google DNS
			"1.1.1.1:53", // CloudFlare DNS
		},
		FallbackServers: []string{
			"208.67.222.222:53", // OpenDNS
			"64.6.64.6:53",      // Verisign
		},
		Timeout:       5 * time.Second,
		CacheDuration: 5 * time.Minute,
	}
}

// DNSChecker manages DNS resolution with fallback support
type DNSChecker struct {
	config         DNSConfig
	customResolver *net.Resolver
	cache          map[string][]string
	cacheTime      map[string]time.Time
	mu             sync.RWMutex
	useFallback    bool
}

// NewDNSChecker creates a new DNS checker instance
func NewDNSChecker(config DNSConfig) *DNSChecker {
	if len(config.PrimaryServers) == 0 {
		config = DefaultDNSConfig()
	}

	return &DNSChecker{
		config:    config,
		cache:     make(map[string][]string),
		cacheTime: make(map[string]time.Time),
	}
}

// CheckAndResolve performs DNS check and returns a resolver
func (d *DNSChecker) CheckAndResolve() *net.Resolver {
	// Test system DNS first
	if d.testSystemDNS() {
		log.TLogln("System DNS check passed")
		return net.DefaultResolver
	}
	log.TLogln("System DNS check failed, using custom resolver")
	d.initCustomResolver()
	return d.customResolver
}

// testSystemDNS checks if system DNS is working properly
func (d *DNSChecker) testSystemDNS() bool {
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, "themoviedb.org")
	if err != nil {
		log.TLogln("DNS lookup error:", err)
		return false
	}

	if len(addrs) == 0 {
		log.TLogln("DNS lookup returned no addresses")
		return false
	}

	// Check for suspicious addresses (DNS hijacking/pollution)
	for _, addr := range addrs {
		if isSuspiciousAddress(addr) {
			log.TLogln("Suspicious DNS address detected:", addr)
			return false
		}
	}

	return true
}

// isSuspiciousAddress checks if an address indicates DNS issues
func isSuspiciousAddress(addr string) bool {
	suspiciousPrefixes := []string{
		"127.0.0.1", // Localhost
		"0.0.0.0",   // Invalid address
		"::1",       // IPv6 localhost
		// "10.",       // Private network
		"192.168.", // Private network
		"169.254.", // Link-local
	}

	for _, prefix := range suspiciousPrefixes {
		if strings.HasPrefix(addr, prefix) {
			return true
		}
	}

	return false
}

// literalAddrResolver is set on the Dialers that reach a DNS server by IP:port. Their
// address needs no lookup, but a Dialer with a nil Resolver reads the global
// net.DefaultResolver on every dial, and dnsResolve replaces that global right after
// pickServer returns while the lookups it abandoned may still be dialling. A non-nil
// Resolver keeps those dials off the global.
var literalAddrResolver = &net.Resolver{}

// initCustomResolver creates a custom resolver with fallback support
func (d *DNSChecker) initCustomResolver() {
	server := d.pickServer()
	if server == "" {
		log.TLogln("No DNS server from the list answered, keeping system resolver")
		d.customResolver = net.DefaultResolver
		return
	}
	log.TLogln("Using DNS server:", server)

	d.customResolver = &net.Resolver{
		PreferGo: true, // Use Go's DNS implementation
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   d.config.Timeout,
				KeepAlive: 30 * time.Second,
				Resolver:  literalAddrResolver,
			}
			// UDP dial succeeds even when nobody answers, so the server
			// is chosen by a real reply in pickServer, not here
			return dialer.DialContext(ctx, network, server)
		},
	}

	d.useFallback = true
}

// LookupHostWithFallback performs DNS lookup with automatic fallback
func (d *DNSChecker) LookupHostWithFallback(host string) ([]string, error) {
	// Check cache first
	if addrs, ok := d.getFromCache(host); ok {
		return addrs, nil
	}

	// Use appropriate resolver
	var resolver *net.Resolver
	if d.useFallback && d.customResolver != nil {
		resolver = d.customResolver
	} else {
		resolver = net.DefaultResolver
	}

	// Perform lookup with timeout
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	addrs, err := resolver.LookupHost(ctx, host)
	if err != nil {
		// If using system DNS fails, try custom resolver
		if !d.useFallback && d.customResolver != nil {
			log.TLogln("System DNS failed, trying custom resolver")
			addrs, err = d.customResolver.LookupHost(ctx, host)
		}
	}

	// Cache successful results
	if err == nil && len(addrs) > 0 {
		d.addToCache(host, addrs)
	}

	return addrs, err
}

// getFromCache retrieves DNS results from cache.
// Expired entries are dropped under the write lock: deleting from a map while
// holding RLock races with every other reader that also sees an expiry.
func (d *DNSChecker) getFromCache(host string) ([]string, bool) {
	d.mu.RLock()
	addrs, ok := d.cache[host]
	fresh := ok && time.Since(d.cacheTime[host]) < d.config.CacheDuration
	var out []string
	if fresh {
		out = append([]string(nil), addrs...)
	}
	d.mu.RUnlock()
	if fresh {
		return out, true
	}
	if !ok {
		return nil, false
	}

	d.mu.Lock()
	if t, still := d.cacheTime[host]; still && time.Since(t) >= d.config.CacheDuration {
		delete(d.cache, host)
		delete(d.cacheTime, host)
	}
	d.mu.Unlock()
	return nil, false
}

// addToCache adds DNS results to cache
func (d *DNSChecker) addToCache(host string, addrs []string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[host] = append([]string(nil), addrs...)
	d.cacheTime[host] = time.Now()
}

// Simple usage function (backward compatible)
func dnsResolve() {
	checker := NewDNSChecker(DefaultDNSConfig())
	resolver := checker.CheckAndResolve()

	// Store the resolver for later use if needed
	net.DefaultResolver = resolver // Optional: replace global resolver

	if !checker.useFallback {
		return // system DNS has just resolved this name in the check
	}
	// Test the resolver
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := resolver.LookupHost(ctx, "themoviedb.org")
	if err != nil {
		log.TLogln("DNS resolution test failed:", err)
	} else {
		log.TLogln("DNS resolution successful, addresses:", addrs)
	}
}

// pickServer asks all servers at once and returns the first one that really
// answered with a non-suspicious address, or "" if none did within Timeout
func (d *DNSChecker) pickServer() string {
	servers := append(append([]string{}, d.config.PrimaryServers...), d.config.FallbackServers...)
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	answered := make(chan string, len(servers))
	for _, server := range servers {
		go func(server string) {
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
					dialer := net.Dialer{Resolver: literalAddrResolver}
					return dialer.DialContext(ctx, network, server)
				},
			}
			addrs, err := r.LookupHost(ctx, "themoviedb.org")
			if err != nil || len(addrs) == 0 {
				return
			}
			for _, addr := range addrs {
				if isSuspiciousAddress(addr) {
					return
				}
			}
			answered <- server
		}(server)
	}

	select {
	case server := <-answered:
		return server
	case <-ctx.Done():
		return ""
	}
}
