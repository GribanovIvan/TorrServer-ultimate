//go:build slim || full

package main

type bindArgs struct {
	IPs []string `arg:"-i,--ip,separate" help:"web server bind addr, or unix:/path/to.sock for a Unix socket (repeatable; default empty binds all interfaces)"`
}
