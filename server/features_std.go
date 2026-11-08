//go:build !slim

package server

import (
	"errors"
	"fmt"
	"strconv"

	"server/log"
	"server/netbind"
	"server/settings"
	"server/tgbot"
)

// Startup of the default and full profiles beyond the core: HTTPS ports and the
// Telegram bot. The slim profile replaces this file with features_slim.go.

// prepareSSL applies --ssl, --sslport, --sslcert and --sslkey and checks that
// the HTTPS port is free.
func prepareSSL() error {
	if !settings.Args.Ssl {
		return nil
	}
	// set settings ssl enabled
	settings.Ssl = settings.Args.Ssl
	if settings.Args.SslPort == "" {
		dbSSlPort := strconv.Itoa(settings.BTsets.SslPort)
		if dbSSlPort != "0" {
			settings.Args.SslPort = dbSSlPort
			log.TLogln("HTTPS port", dbSSlPort, "(from settings; override with --sslport)")
		} else {
			settings.Args.SslPort = settings.DefaultSslPort
			log.TLogln("HTTPS port", settings.DefaultSslPort, "(default)")
		}
	} else { // store ssl port from params to DB
		dbSSlPort, err := strconv.Atoi(settings.Args.SslPort)
		if err == nil {
			settings.BTsets.SslPort = dbSSlPort
		}
		log.TLogln("HTTPS port", settings.Args.SslPort, "(from --sslport, saved to settings)")
	}
	// check if ssl cert and key files exist
	if settings.Args.SslCert != "" && settings.Args.SslKey != "" {
		// set settings ssl cert and key files
		settings.BTsets.SslCert = settings.Args.SslCert
		settings.BTsets.SslKey = settings.Args.SslKey
	}
	log.TLogln("Check web ssl port", settings.Args.SslPort)
	if err := netbind.CheckPort(netbind.TCPHosts(settings.Args.IPs), settings.Args.SslPort); err != nil {
		msg := fmt.Sprintf("port %s already in use! Please set different ssl port for HTTPS", settings.Args.SslPort)
		log.TLogln(msg)
		return errors.New(msg)
	}
	return nil
}

func startTelegramBot() {
	if settings.Args.TGToken != "" {
		if err := tgbot.Start(settings.Args.TGToken); err != nil {
			log.TLogln("tg bot start failed", err)
		}
	}
}
