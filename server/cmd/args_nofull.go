//go:build !full

package main

import "server/settings"

// The default profile has no full-only flags (args_full.go).
type fullArgs struct{}

func (fullArgs) apply(*settings.ExecArgs) {}
func (fullArgs) start()                   {}
