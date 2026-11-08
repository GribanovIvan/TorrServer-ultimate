//go:build !slim

package web

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"server/bonjour"
	"server/dlna"
	gstreamer "server/gstreamer/bridge"
	"server/log"
	"server/mcp"
	"server/netbind"
	"server/rutor"
	"server/settings"
	"server/torrfs/fuse"
	"server/torrfs/webdav"
	"server/web/auth"
	"server/web/msx"
	"server/web/pages"
	"server/web/sslcerts"
	"server/web/waf"
)

// Features of the default and full profiles. The slim profile replaces this
// file with features_slim.go, so none of these packages is linked there.

// stopRenew stops the self-signed cert renewal loop.
var stopRenew chan struct{}

// featureHeaders are the extra CORS request headers: Telegram Mini App auth and MCP.
var featureHeaders = []string{
	"X-Telegram-Init-Data",
	"Mcp-Protocol-Version", "Mcp-Session-Id", "Last-Event-ID", "Mcp-Method", "Mcp-Name",
}

// prepareTLS makes sure an HTTPS certificate exists before anything listens.
func prepareTLS() error {
	if !settings.Ssl {
		return nil
	}
	cert, key, changed, err := sslcerts.EnsureCert(settings.BTsets.SslCert, settings.BTsets.SslKey, certIPs())
	if err != nil {
		log.TLogln("Cannot start HTTPS (fix --sslcert/--sslkey or clear them in settings to use a self-signed cert):", err)
		return err
	}
	if changed {
		settings.BTsets.SslCert, settings.BTsets.SslKey = cert, key
		log.TLogln("Saving path to ssl cert and key in db", cert, key)
		settings.SetBTSets(settings.BTsets)
	}
	return nil
}

func startServices() {
	rutor.Start()
}

func firewall() []gin.HandlerFunc {
	return []gin.HandlerFunc{waf.WAF()}
}

func setupFeatures(route *gin.Engine) {
	setupSSLRoutes(route)
	mcp.Mount(route.Group("/", auth.CheckAuth()))
	gstreamer.SetupRoute(route)
	msx.SetupRoute(route)
	pages.SetupRoute(route)
	if settings.Args.WebDAV {
		webdav.MountWebDAV(route)
	}

	if settings.BTsets.EnableDLNA {
		if netbind.HasTCP(settings.IPs) {
			dlna.Start()
		} else {
			log.TLogln("DLNA is off: no TCP --ip to serve it on (Unix socket only)")
		}
	}
	if settings.BTsets.EnableBonjour {
		if netbind.HasTCP(settings.IPs) {
			bonjour.Start()
		} else {
			log.TLogln("Bonjour is off: no TCP --ip to advertise (Unix socket only)")
		}
	}

	// Auto-mount FUSE filesystem if enabled
	fuse.FuseAutoMount()

	// /swagger and /swagger/ → index.html. Do not register GET "/swagger/" separately:
	// in gin it conflicts with "/swagger/*any" (empty segment vs catch-all).
	route.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/index.html")
	})
	swagger := swaggerHandler()
	route.GET("/swagger/*any", func(c *gin.Context) {
		if c.Param("any") == "/" {
			c.Redirect(http.StatusFound, "/swagger/index.html")
			return
		}
		swagger(c)
	})
}

func stopFeatures() {
	gstreamer.Stop()
	dlna.Stop()
	bonjour.Stop()
	// Unmount FUSE filesystem if mounted
	fuse.FuseCleanup()
}

// startTLS binds the HTTPS port with --ssl. Plain HTTP sent to it is redirected
// to https://.
func startTLS(h http.Handler) error {
	tcpSsl := settings.Ssl && netbind.HasTCP(settings.IPs)
	if settings.Ssl && !tcpSsl {
		log.TLogln("--ssl: no TCP --ip to bind, HTTPS is not opened (Unix socket serves plain HTTP)")
	}
	if !tcpSsl {
		return nil
	}
	loader, err := sslcerts.NewLoader(sslCertPaths)
	if err != nil {
		return fmt.Errorf("load ssl cert: %w", err)
	}
	ln, err := netbind.Listen(netbind.TCPHosts(settings.IPs), settings.SslPort)
	if err != nil {
		return fmt.Errorf("https listen: %w", err)
	}
	tlsLn, plainLn := splitTLS(ln)

	srv := newServer(h)
	// MinVersion is left to Go's default (TLS 1.2) so GODEBUG=tls10server=1 still works for old TVs
	srv.TLSConfig = &tls.Config{GetCertificate: loader.GetCertificate}
	serve(func() error { return srv.ServeTLS(tlsLn, "", "") })

	redirect := newServer(httpsRedirectHandler())
	serve(func() error { return redirect.Serve(plainLn) })
	logAddrs("Start https server at", settings.SslPort)

	stopRenew = make(chan struct{})
	go sslcerts.RenewLoop(stopRenew, time.Hour, sslCertPaths, certIPs)
	return nil
}

func stopTLS() {
	if stopRenew != nil {
		close(stopRenew)
		stopRenew = nil
	}
}

// startInternalServer serves h over plain HTTP on a random loopback port for
// TorrServer's requests to itself (ffprobe, GStreamer). Unlike the public HTTP port it is
// never redirected by --force-https, and it works when --ip excludes loopback. Proxies
// and port mappings can't target it, as the port changes on every start.
func startInternalServer(h http.Handler) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.TLogln("Internal loopback listener unavailable, self-requests use the public ports:", err)
		return
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	srv := newServer(h)
	serve(func() error { return srv.Serve(ln) })
	settings.InternalPort = port
	log.TLogln("Internal loopback listener at", ln.Addr(), "(TorrServer's own requests only)")
}

func warnHTTPSOnly() {
	warnSelfSignedStrict("--https-only")
}

// forceHTTPS returns the handler for the plain HTTP port with --force-https, or
// nil when that port serves h as is.
func forceHTTPS(h http.Handler) http.Handler {
	if settings.Args == nil || !settings.Args.ForceHTTPS || !settings.Ssl {
		return nil
	}
	httpMedia := settings.Args.HTTPMedia
	if httpMedia {
		logAddrs("Start http server (media only, everything else redirects to https) at", settings.Port)
		log.TLogln("Warning: --http-media serves stream URLs over plain HTTP; they and Basic auth " +
			"credentials travel unencrypted. Don't expose the HTTP port to the internet.")
	} else {
		logAddrs("Start http server (redirect to https) at", settings.Port)
		warnSelfSignedStrict("--force-https")
	}
	return forceHTTPSHandler(h, httpMedia)
}

// warnSelfSignedStrict warns when media is only served over HTTPS with the self-signed
// certificate, which most players and TVs reject.
func warnSelfSignedStrict(flag string) {
	if sslcerts.IsGenerated(sslCertPaths()) {
		log.TLogln("Warning: " + flag + " with a self-signed certificate: media players and TVs " +
			"usually reject it and won't play. Use a trusted certificate (see docs/https.md)" +
			" or, on a trusted network, --force-https --http-media.")
	}
}

// certIPs are the addresses the self-signed cert must cover: the bound IPs when
// --ip is given, all local IPs otherwise.
func certIPs() []string {
	var bound []string
	for _, ip := range settings.IPs {
		if netbind.IsUnix(ip) {
			continue
		}
		if parsed := net.ParseIP(ip); parsed != nil && !parsed.IsUnspecified() {
			bound = append(bound, ip)
		} else {
			return GetLocalIps() // "", 0.0.0.0 or :: bind everything
		}
	}
	if len(bound) == 0 {
		return GetLocalIps()
	}
	return bound
}

func sslCertPaths() (string, string) {
	return settings.BTsets.SslCert, settings.BTsets.SslKey
}
