//go:build slim

package main

import "server/settings"

// The slim profile has only the core flags of main.go: no HTTPS, web UI,
// torrent dir watch, search, Telegram bot, FUSE or WebDAV flags.
type featureArgs struct{}

func (featureArgs) apply(*settings.ExecArgs) {}
func (featureArgs) check() error             { return nil }
func (featureArgs) start(string)             {}
