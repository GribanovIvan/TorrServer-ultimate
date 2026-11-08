//go:build full

package main

import (
	"testing"

	"github.com/alexflint/go-arg"
)

func TestFullArgs(t *testing.T) {
	var params args
	p, err := arg.NewParser(arg.Config{}, &params)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Parse([]string{"--pprof", "--downloaddir", "/srv/dl", "--ip", "unix:/run/ts.sock"}); err != nil {
		t.Fatal(err)
	}
	if !params.Pprof || params.DownloadDir != "/srv/dl" || len(params.IPs) != 1 {
		t.Fatalf("%+v", params)
	}
}
