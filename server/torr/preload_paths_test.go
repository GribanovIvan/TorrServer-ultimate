package torr

import (
	"context"
	"errors"
	"testing"
	"time"

	"server/torr/offlinetest"
	"server/torr/state"
)

func TestPreloadSpansClamps(t *testing.T) {
	tests := []struct {
		name             string
		size, file, edge int64
		head, tail       int64
	}{
		{"negative file length", 10, -5, 4, 0, 0},
		{"negative budget", -3, 100, 4, 0, 96},
		{"budget past the file", 500, 100, 4, 96, 96},
		{"edge bigger than the file", 50, 100, 200, 50, 50},
	}
	for _, tc := range tests {
		head, tail := preloadSpans(tc.size, tc.file, tc.edge)
		if head != tc.head || tail != tc.tail {
			t.Errorf("%s: got (%d,%d), want (%d,%d)", tc.name, head, tail, tc.head, tc.tail)
		}
	}
}

func TestBindStreamReadContextHonorsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bound := bindStreamReadContext(ctx, stallReadSeeker{})
	if _, ok := bound.(streamCtxReader); !ok {
		t.Fatalf("source with ReadContext not wrapped: %T", bound)
	}
	errc := make(chan error, 1)
	go func() {
		_, err := bound.Read(make([]byte, 8))
		errc <- err
	}()
	cancel()
	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("read after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read ignored the canceled context")
	}
}

func setStat(tor *Torrent, s state.TorrentStat) {
	tor.muTorrent.Lock()
	tor.Stat = s
	tor.muTorrent.Unlock()
}

func statOf(tor *Torrent) state.TorrentStat {
	tor.muTorrent.Lock()
	defer tor.muTorrent.Unlock()
	return tor.Stat
}

func waitStat(t *testing.T, tor *Torrent, want state.TorrentStat) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for statOf(tor) != want {
		if time.Now().After(deadline) {
			t.Fatalf("stat %v, want %v", statOf(tor), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func runPreload(tor *Torrent, index int, size int64) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		tor.Preload(index, size)
	}()
	return done
}

func waitDone(t *testing.T, done <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("%s: Preload did not return", what)
	}
}

func TestPreloadSkipsTorrentThatIsNotWorking(t *testing.T) {
	tor, _ := connectShow(t)

	// Still "getting info" after the wait: Preload gives up without a state change.
	setStat(tor, state.TorrentGettingInfo)
	tor.Preload(1, 4<<10)
	if got := statOf(tor); got != state.TorrentGettingInfo {
		t.Fatalf("stat %v", got)
	}
	setStat(tor, state.TorrentWorking)

	// "drop all torrents" nils the client torrent but leaves Stat as is.
	tor.muTorrent.Lock()
	saved := tor.Torrent
	tor.Torrent = nil
	tor.muTorrent.Unlock()
	tor.Preload(1, 4<<10)
	if got := statOf(tor); got != state.TorrentWorking {
		t.Fatalf("dropped torrent changed stat to %v", got)
	}
	tor.muTorrent.Lock()
	tor.Torrent = saved
	tor.muTorrent.Unlock()
}

func TestPreloadWaitInfoFailsOnDroppedTorrent(t *testing.T) {
	tor, _ := connectShow(t)
	tor.muTorrent.Lock()
	saved := tor.Torrent
	tor.Torrent = nil
	tor.Stat = state.TorrentGettingInfo
	tor.muTorrent.Unlock()
	tor.Preload(1, 4<<10)
	tor.muTorrent.Lock()
	tor.Torrent = saved
	tor.Stat = state.TorrentWorking
	tor.muTorrent.Unlock()
}

func TestPreloadUnknownIndexUsesFirstFile(t *testing.T) {
	tor, _ := connectShow(t)
	// Index 99 matches no FileStat; the first file is read, and a budget past the
	// file length is clamped to it.
	tor.Preload(99, 1<<30)
	if got := statOf(tor); got != state.TorrentWorking {
		t.Fatalf("stat %v", got)
	}
	if tor.GetCache() == nil || tor.GetCache().GetState().Filled == 0 {
		t.Fatal("first file was not read")
	}
	if tor.findFileIndex(tor.Torrent, 99) != nil {
		t.Fatal("findFileIndex found an unknown id")
	}
}

// A preload that cannot get its bytes stays in TorrentPreload and ends as soon
// as the torrent is closed. It lives past one second so the progress logger ticks.
func TestPreloadCancelledWithoutPeers(t *testing.T) {
	show := offlinetest.WriteSized(t, "movie.mkv", 48<<20, 1<<20)
	offlinetest.Prepare(t)
	bt := NewBTS()
	InitApiHelper(bt)
	if err := bt.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(bt.Disconnect)
	tor, err := AddTorrent(show.Spec, "Movie", "", "", "movie")
	if err != nil || !tor.GotInfo() {
		t.Fatalf("AddTorrent: %v", err)
	}
	// No peers: the head and tail readers block on the first piece.
	done := runPreload(tor, 1, 32<<20)
	waitStat(t, tor, state.TorrentPreload)
	time.Sleep(1300 * time.Millisecond) // one logger tick while still preloading
	tor.Close()
	waitDone(t, done, "cancel without peers")
	if got := statOf(tor); got != state.TorrentClosed {
		t.Fatalf("stat %v, want closed", got)
	}
}
