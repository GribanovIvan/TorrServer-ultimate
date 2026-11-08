//go:build slim

package server

import (
	"strings"
	"testing"

	"server/settings"
)

func TestStartRejectsUnbindablePort(t *testing.T) {
	dir := t.TempDir()
	oldArgs, oldPath, oldSets := settings.Args, settings.Path, settings.BTsets
	t.Cleanup(func() {
		settings.CloseDB()
		settings.Args, settings.Path, settings.BTsets = oldArgs, oldPath, oldSets
	})
	settings.Path = dir
	settings.Args = &settings.ExecArgs{Path: dir, Port: "1", IPs: []string{"127.0.0.1"}}
	if err := Start(); err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("http: %v", err)
	}
}
