package web

import (
	"fmt"
	"net"
	"net/http"
	"sort"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/location/v2"
	"github.com/gin-gonic/gin"
	"github.com/wlynxg/anet"

	"server/log"
	"server/netbind"
	"server/settings"
	"server/torr"
	"server/version"
	"server/web/api"
	"server/web/auth"
)

var (
	BTS      = torr.NewBTS()
	waitChan = make(chan error)
	// listIfaces and ifaceAddrs are the host lookups behind GetLocalIps.
	// Tests replace them; production uses anet.
	listIfaces = anet.Interfaces
	ifaceAddrs = anet.InterfaceAddrsByInterface
)

//	@title			Swagger Torrserver API
//	@version		{version.Version}
//	@description	Torrent streaming server.

//	@license.name	GPL 3.0

//	@BasePath	/

//	@securityDefinitions.basic	BasicAuth

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func Start() error {
	log.TLogln("Start TorrServer " + version.Full() + " torrent " + version.GetTorrentVersion())
	ips := GetLocalIps()
	if len(ips) > 0 {
		log.TLogln("Local IPs:", ips)
	}
	if err := prepareTLS(); err != nil {
		return err
	}
	log.TLogln("Client config:", settings.BTsets)
	err := BTS.Connect()
	if err != nil {
		log.TLogln("BTS.Connect() error!", err)
		return err
	}
	startServices()

	gin.SetMode(gin.ReleaseMode)

	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AllowPrivateNetwork = true
	corsCfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "HEAD", "OPTIONS", "DELETE"}
	corsCfg.AllowHeaders = append([]string{
		"Origin", "Content-Length", "Content-Type", "X-Requested-With",
		"Accept", "Authorization",
	}, featureHeaders...)

	route := gin.New()
	route.Use(log.WebLogger())
	route.Use(firewall()...)
	route.Use(gin.Recovery(), cors.New(corsCfg), location.Default())
	auth.SetupAuth(route)

	route.GET("/echo", echo)

	api.SetupRoute(route)
	setupFeatures(route)

	if err := startServers(route); err != nil {
		shutdownServers()
		return err
	}
	return nil
}

// startServers binds the HTTP port (and, in builds with HTTPS, the HTTPS one)
// and starts serving h.
func startServers(h http.Handler) error {
	if err := startTLS(h); err != nil {
		return err
	}

	startInternalServer(h)

	if !settings.HTTPEnabled() {
		log.TLogln("HTTPS only: the plain HTTP port", settings.Port, "is not opened")
		warnHTTPSOnly()
		return nil
	}

	ln, err := netbind.Listen(settings.IPs, settings.Port)
	if err != nil {
		return fmt.Errorf("http listen: %w", err)
	}
	if redirect := forceHTTPS(h); redirect != nil {
		srv := newServer(redirect)
		serve(func() error { return srv.Serve(ln) })
		return nil
	}
	srv := newServer(h)
	serve(func() error { return srv.Serve(ln) })
	logAddrs("Start http server at", settings.Port)
	return nil
}

func logAddrs(msg, port string) {
	for _, ip := range netbind.Normalize(settings.IPs) {
		log.TLogln(msg, netbind.Addr(ip, port))
	}
}

func Wait() error {
	return <-waitChan
}

func Stop() {
	// stop accepting requests before tearing down what they depend on
	stopTLS()
	shutdownServers()
	settings.InternalPort = ""
	stopFeatures()
	BTS.Disconnect()
	waitChan <- nil
}

// echo godoc
//
//	@Summary		Tests server status
//	@Description	Tests whether server is alive or not
//
//	@Tags			API
//
//	@Produce		plain
//	@Success		200	{string}	string	"Server version"
//	@Router			/echo [get]
func echo(c *gin.Context) {
	c.String(200, "%v", version.Version)
}

func GetLocalIps() []string {
	ifaces, err := listIfaces()
	if err != nil {
		log.TLogln("Error get local IPs")
		return nil
	}
	var list []string
	for _, i := range ifaces {
		addrs, _ := ifaceAddrs(&i)
		if i.Flags&net.FlagUp == net.FlagUp {
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() {
					list = append(list, ip.String())
				}
			}
		}
	}
	sort.Strings(list)
	return list
}
