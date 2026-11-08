package api

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/gin-gonic/gin"

	"server/torr"
	"server/torr/state"
)

func TestStatusFromSpecMatchesTorrentFileIDs(t *testing.T) {
	info := metainfo.Info{
		Name:        "Show",
		PieceLength: 16 << 10,
		Files: []metainfo.FileInfo{
			{Path: []string{"Episode 10.mp4"}, Length: 1},
			{Path: []string{"Episode 2.mp4"}, Length: 1},
			{Path: []string{"Episode 3.mp4"}, Length: 1},
		},
	}
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	tr := &torr.Torrent{TorrentSpec: &torrent.TorrentSpec{InfoBytes: infoBytes}}

	// Status numbers files by their library path sorted with utils.CompareStrings,
	// and stream links resolve index= against those ids.
	want := []string{"Show/Episode 2.mp4", "Show/Episode 3.mp4", "Show/Episode 10.mp4"}
	st := statusFromSpec(tr)
	if st == nil || len(st.FileStats) != len(want) {
		t.Fatalf("statusFromSpec = %+v", st)
	}
	for i, f := range st.FileStats {
		if f.Id != i+1 || f.Path != want[i] {
			t.Errorf("file %d = {Id: %d, Path: %q}, want {Id: %d, Path: %q}", i, f.Id, f.Path, i+1, want[i])
		}
	}
}

func TestSafeAttachmentName(t *testing.T) {
	if got := safeAttachmentName("Show.m3u"); got != "Show.m3u" {
		t.Fatalf("plain name %q", got)
	}
	if got := safeAttachmentName("a\"\r\nX: 1.m3u"); got != "aX: 1.m3u" {
		t.Fatalf("stripped %q", got)
	}
	if got := safeAttachmentName("  \r\n"); got != "playlist.m3u" {
		t.Fatalf("empty %q", got)
	}
}

func TestFindFileNamesakesNeedsTheWholeName(t *testing.T) {
	paths := []string{
		"Show/Show S01E1.mkv",
		"Show/Show S01E10.mkv",
		"Show/Show S01E11.mkv",
		"Show/Show S01E1.rus.mka",
		"Show/Sound/Show S01E1 [Studio].mka",
		"Show/Subs/Show S01E1/English.srt",
		"Show/Фильм.mkv",
		"Show/Фильмы.mkv",
		"Show/Фильм.ru.srt",
	}
	var files []*state.TorrentFileStat
	for i, p := range paths {
		files = append(files, &state.TorrentFileStat{Id: i + 1, Path: p})
	}
	names := func(video int) []string {
		var got []string
		for _, f := range findFileNamesakes(files, files[video]) {
			got = append(got, f.Path)
		}
		return got
	}

	want := []string{"Show/Show S01E1.rus.mka", "Show/Sound/Show S01E1 [Studio].mka", "Show/Subs/Show S01E1/English.srt"}
	if got := names(0); !slices.Equal(got, want) {
		t.Errorf("S01E1 namesakes = %q, want %q", got, want)
	}
	if got := names(1); len(got) != 0 {
		t.Errorf("S01E10 namesakes = %q, want none", got)
	}
	if got := names(6); !slices.Equal(got, []string{"Show/Фильм.ru.srt"}) {
		t.Errorf("Фильм namesakes = %q", got)
	}
}

func TestSendM3UETagFollowsBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// gin buffers the status, so read it from the context writer.
	send := func(m3u, ifNoneMatch string) (int, http.Header, string) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/playlist", nil)
		if ifNoneMatch != "" {
			c.Request.Header.Set("If-None-Match", ifNoneMatch)
		}
		sendM3U(c, "movie.m3u", "08ada5a7", m3u)
		return c.Writer.Status(), w.Header(), w.Body.String()
	}

	_, header, _ := send("#EXTM3U\nhttp://h/stream/a.mkv\n", "")
	tag := header.Get("ETag")
	if tag == "" {
		t.Fatal("no ETag")
	}
	if status, _, _ := send("#EXTM3U\nhttp://h/stream/a.mkv\n", tag); status != http.StatusNotModified {
		t.Fatalf("same playlist: status = %d, want 304", status)
	}
	status, _, body := send("#EXTM3U\nhttp://h/stream/b.mkv\n", tag)
	if status != http.StatusOK || !strings.Contains(body, "b.mkv") {
		t.Fatalf("changed playlist with the old ETag: status = %d, body %q, want 200 with the new list", status, body)
	}
}
