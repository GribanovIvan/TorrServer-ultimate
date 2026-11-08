//go:build !slim && !full

package main

import (
	"testing"

	"github.com/alexflint/go-arg"
)

// TestDefaultHasNoFullFlags: --pprof and --downloaddir are in the full profile only.
func TestDefaultHasNoFullFlags(t *testing.T) {
	for _, argv := range [][]string{{"--pprof"}, {"--downloaddir", "/srv/dl"}, {"-x", "/srv/dl"}} {
		var params args
		p, err := arg.NewParser(arg.Config{}, &params)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Parse(argv); err == nil {
			t.Fatalf("%v accepted", argv)
		}
	}
}
