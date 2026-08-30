package torr

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"server/ffprobe"

	"server/log"
	"server/settings"
	"server/torr/state"
	utils2 "server/utils"

	"github.com/anacrolix/torrent"
)

func (t *Torrent) Preload(index int, size int64) {
	if size <= 0 {
		return
	}
	t.PreloadSize = size

	if t.Stat == state.TorrentGettingInfo {
		if !t.WaitInfo() {
			return
		}
		// wait change status
		time.Sleep(100 * time.Millisecond)
	}

	t.muTorrent.Lock()
	if t.Stat != state.TorrentWorking {
		t.muTorrent.Unlock()
		return
	}

	t.Stat = state.TorrentPreload
	t.muTorrent.Unlock()

	t.dropOthers()

	defer func() {
		t.muTorrent.Lock()
		if t.Stat == state.TorrentPreload {
			t.Stat = state.TorrentWorking
		}
		t.muTorrent.Unlock()
		// Очистка по окончании прелоада
		t.BitRate = ""
		t.DurationSeconds = 0
	}()

	file := t.findFileIndex(index)
	if file == nil {
		file = t.Files()[0]
	}

	if size > file.Length() {
		size = file.Length()
	}

	if t.Info() == nil {
		return
	}
	pieceLength := t.Info().PieceLength
	// Current in-flight window of the start reader, exposed to the progress log.
	var windowRA atomic.Int64

	timeout := time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout)
	if timeout > time.Minute {
		timeout = time.Minute
	}

	// Create a stop channel for the logging goroutine
	logStopChan := make(chan struct{})
	defer close(logStopChan) // Ensure logging stops when function returns

	// Запуск лога в отдельном потоке
	go func(stopChan <-chan struct{}) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.muTorrent.Lock()
				stat := t.Stat
				t.muTorrent.Unlock()

				if stat != state.TorrentPreload {
					return
				}

				statStr := fmt.Sprint(file.Torrent().InfoHash().HexString(), " ",
					utils2.Format(float64(t.PreloadedBytes)), "/",
					utils2.Format(float64(t.PreloadSize)), " Speed:",
					utils2.Format(t.DownloadSpeed), " Peers:",
					t.Torrent.Stats().ActivePeers, "/",
					t.Torrent.Stats().TotalPeers, " [Seeds:",
					t.Torrent.Stats().ConnectedSeeders, "] RA:",
					utils2.Format(float64(windowRA.Load())), " Piece:",
					utils2.Format(float64(pieceLength)))
				log.TLogln("Preload:", statStr)
				t.AddExpiredTime(timeout)
			case <-stopChan:
				return
			}
		}
	}(logStopChan)

	if ffprobe.Exists() {
		link := "http://127.0.0.1:" + settings.Port + "/play/" + t.Hash().HexString() + "/" + strconv.Itoa(index)
		if settings.Ssl {
			link = "https://127.0.0.1:" + settings.SslPort + "/play/" + t.Hash().HexString() + "/" + strconv.Itoa(index)
		}
		if data, err := ffprobe.ProbeUrl(link); err == nil {
			t.BitRate = data.Format.BitRate
			t.DurationSeconds = data.Format.DurationSeconds
		}
	}

	// Check if torrent was closed
	t.muTorrent.Lock()
	isClosed := t.Stat == state.TorrentClosed
	t.muTorrent.Unlock()

	if isClosed {
		log.TLogln("End preload: torrent closed")
		return
	}

	// startend -> 8/16 MB
	startend := pieceLength
	if startend < 8<<20 {
		startend = 8 << 20
	}

	readerStart := file.NewReader()
	if readerStart == nil {
		log.TLogln("End preload: null reader")
		return
	}
	defer readerStart.Close()

	readerStart.SetResponsive()
	readerStart.SetReadahead(0)
	readerStartEnd := size - startend

	if readerStartEnd < 0 {
		// Если конец начального ридера оказался за началом
		readerStartEnd = size
	}
	if readerStartEnd > file.Length() {
		// Если конец начального ридера оказался после конца файла
		readerStartEnd = file.Length()
	}

	readerEndStart := file.Length() - startend
	readerEndEnd := file.Length()

	var wg sync.WaitGroup
	var preloadErr error

	// Start end range preload if needed
	if readerEndStart > readerStartEnd {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Check if we should still preload
			t.muTorrent.Lock()
			shouldPreload := t.Stat == state.TorrentPreload
			t.muTorrent.Unlock()

			if !shouldPreload {
				return
			}

			readerEnd := file.NewReader()
			if readerEnd == nil {
				log.TLogln("Err preload: null reader")
				preloadErr = fmt.Errorf("null reader for end range")
				return
			}
			defer readerEnd.Close() // Ensure reader is always closed

			readerEnd.SetResponsive()
			// The whole end chunk is the reader's span; requesting it at once
			// avoids one-piece-at-a-time serialization on the same peers the
			// start reader is using.
			readerEnd.SetReadahead(preloadWindow(size, startend, pieceLength))

			_, err := readerEnd.Seek(readerEndStart, io.SeekStart)
			if err != nil {
				log.TLogln("Err preload seek:", err)
				preloadErr = err
				return
			}

			offset := readerEndStart
			tmp := make([]byte, 32768)
			for offset+int64(len(tmp)) < readerEndEnd {
				n, err := readerEnd.Read(tmp)
				if err != nil {
					if err != io.EOF {
						log.TLogln("Err preload read:", err)
						preloadErr = err
					}
					break
				}
				offset += int64(n)

				// Check if we should continue
				t.muTorrent.Lock()
				shouldContinue := t.Stat == state.TorrentPreload
				t.muTorrent.Unlock()

				if !shouldContinue {
					break
				}
			}
		}()
	}

	// Main preload section
	readahead := preloadWindow(size, readerStartEnd, pieceLength)
	readerStart.SetReadahead(readahead)
	windowRA.Store(readahead)

	offset := int64(0)
	tmp := make([]byte, 32768)
	for offset+int64(len(tmp)) < readerStartEnd {
		// Check if we should continue
		t.muTorrent.Lock()
		shouldContinue := t.Stat == state.TorrentPreload
		t.muTorrent.Unlock()

		if !shouldContinue {
			log.TLogln("Preload cancelled")
			break
		}

		n, err := readerStart.Read(tmp)
		if err != nil {
			if err != io.EOF {
				log.TLogln("Error preload:", err)
			}
			break
		}
		offset += int64(n)

		// Shrink the window to what is left so the reader does not request
		// past the preload target, but only in whole-piece steps: every
		// SetReadahead takes the anacrolix client lock, and this loop runs
		// once per 32 KB read. Dropping straight to 0 (the old behaviour)
		// serialized the entire last window one piece at a time.
		if remain := readerStartEnd - offset; remain >= 0 && readahead-remain >= pieceLength {
			readahead = remain
			readerStart.SetReadahead(readahead)
			windowRA.Store(readahead)
		}
	}

	// Wait for end range preload to complete
	wg.Wait()

	// Check if end range preload failed
	if preloadErr != nil {
		log.TLogln("End range preload failed:", preloadErr)
	}

	// Final log
	t.muTorrent.Lock()
	finalStat := t.Stat
	t.muTorrent.Unlock()

	if finalStat == state.TorrentPreload {
		log.TLogln("End preload:", file.Torrent().InfoHash().HexString(),
			"Peers:", t.Torrent.Stats().ActivePeers, "/",
			t.Torrent.Stats().TotalPeers, "[ Seeds:",
			t.Torrent.Stats().ConnectedSeeders, "]")
	}
}

// preloadWindow returns the in-flight readahead for a preload reader.
//
// A fixed pieceLength*4 pins the pipeline to four pieces regardless of link
// speed or configured cache, so the fill rate is capped at four piece
// completions per round trip and a bigger cache only makes the wait longer.
// Sizing the window off the preload target lets it scale with CacheSize and
// PreloadCache, while pieceLength*4 remains the floor for small caches. The
// result is clamped to span (the bytes this reader will actually read) so the
// reader never requests beyond the preload target; raw anacrolix readers do
// not clamp to cache capacity themselves.
func preloadWindow(size, span, pieceLength int64) int64 {
	ra := size / 4
	if floor := pieceLength * 4; ra < floor {
		ra = floor
	}
	if ra > span {
		ra = span
	}
	return ra
}

func (t *Torrent) findFileIndex(index int) *torrent.File {
	st := t.Status()
	var stFile *state.TorrentFileStat
	for _, f := range st.FileStats {
		if index == f.Id {
			stFile = f
			break
		}
	}
	if stFile == nil {
		return nil
	}
	for _, file := range t.Files() {
		if file.Path() == stFile.Path {
			return file
		}
	}
	return nil
}
