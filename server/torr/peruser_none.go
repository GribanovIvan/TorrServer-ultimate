//go:build !full

package torr

import "github.com/anacrolix/torrent"

// Torrents are shared by all accounts: per-user lists are in the full profile
// (peruser.go).

func AddTorrentForUser(_ string, spec *torrent.TorrentSpec, title, poster, data, category string) (*Torrent, error) {
	return AddTorrent(spec, title, poster, data, category)
}

func RemTorrentForUser(hashHex, _ string) bool {
	RemTorrent(hashHex)
	return true
}

func ListTorrentForUser(string) []*Torrent { return ListTorrent() }
