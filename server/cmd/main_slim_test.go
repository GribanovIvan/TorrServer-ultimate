//go:build slim

package main

import (
	"testing"

	"github.com/alexflint/go-arg"
)

func watchDirArgs(string) []string { return nil }

func checkWatchDirAddsTorrent(*testing.T, string) {}

// TestSlimHasNoFeatureFlags checks that flags of the left-out features are not
// accepted at all.
func TestSlimHasNoFeatureFlags(t *testing.T) {
	for _, flag := range []string{"--ssl", "--webdav", "--pprof", "-t", "-T", "-f", "--force-https", "-u", "-s", "--downloaddir"} {
		var params args
		p, err := arg.NewParser(arg.Config{}, &params)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Parse([]string{flag, "x"}); err == nil {
			t.Fatalf("%s accepted", flag)
		}
	}
}
