//go:build !slim && !full

package main

type bindArgs struct {
	IPs []string `arg:"-i,--ip,separate" help:"web server bind addr (repeatable; default empty binds all interfaces)"`
}
