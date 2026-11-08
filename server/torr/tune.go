//go:build !full

package torr

import "github.com/anacrolix/torrent"

// tuneClient leaves the client config as configure set it. The full profile
// adds the network values of Tommixoft/TorrServer (tune_full.go).
func tuneClient(*torrent.ClientConfig) {}
