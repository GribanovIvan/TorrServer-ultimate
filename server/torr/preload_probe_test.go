//go:build !slim

package torr

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"server/ffprobe"
	"server/torr/state"
)

func mockFFProbe(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("ffprobe mock is a shell script")
	}
	path := filepath.Join(t.TempDir(), "ffprobe")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ffprobe.SetBinPath(path))
}

const probeOK = "#!/bin/sh\nprintf '%s' '{\"format\":{\"duration\":\"1.5\",\"bit_rate\":\"1000\"}}'\n"

func TestPreloadClearsProbeResult(t *testing.T) {
	for name, script := range map[string]string{
		"probe ok":     probeOK,
		"probe failed": "#!/bin/sh\nexit 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			tor, _ := connectShow(t)
			mockFFProbe(t, script) // after Prepare, which points ffprobe at a missing binary
			tor.Preload(1, 8<<10)
			if got := statOf(tor); got != state.TorrentWorking {
				t.Fatalf("stat %v", got)
			}
			if tor.BitRate != "" || tor.DurationSeconds != 0 {
				t.Fatalf("probe result kept after preload: %q %v", tor.BitRate, tor.DurationSeconds)
			}
		})
	}
}

func TestPreloadStopsWhenClosedDuringProbe(t *testing.T) {
	tor, _ := connectShow(t)
	mockFFProbe(t, "#!/bin/sh\nsleep 0.4\n"+probeOK)
	done := runPreload(tor, 1, 8<<10)
	waitStat(t, tor, state.TorrentPreload)
	tor.Close()
	waitDone(t, done, "close during probe")
	if got := statOf(tor); got != state.TorrentClosed {
		t.Fatalf("stat %v, want closed", got)
	}
}
