//go:build full

package torr

import (
	"time"

	"github.com/anacrolix/torrent"

	"server/settings"
	"server/torr/utils"
)

// tuneClient applies the network values of Tommixoft/TorrServer (full profile
// only; the default profile keeps the library and upstream values, tune.go).
func tuneClient(cfg *torrent.ClientConfig) {
	// Keep dialing bursts modest: hundreds of simultaneous half-open
	// connections overwhelm home routers and TV network stacks.
	cfg.TotalHalfOpenConns = 100
	// Seed mode keeps the peer pool warm: the torrent library dials new peers
	// only while needData() || seeding() is true, and a filled stream window
	// has no pending pieces, so without Seed the client stalls at a handful
	// of connections and re-dials from scratch on every window advance.
	// Actual upload stays capped below - enough for choke-algorithm
	// reciprocity, while an uncapped uplink on weak TV Wi-Fi drowns TCP ACKs
	// and destabilizes the download itself.
	cfg.Seed = !settings.BTsets.DisableUpload
	if !settings.BTsets.DisableUpload {
		uploadLimit := settings.BTsets.UploadRateLimit * 1024
		if uploadLimit <= 0 {
			uploadLimit = 128 * 1024
		}
		cfg.UploadRateLimiter = utils.Limit(uploadLimit)
	}
	// Default half-open dial cap (25) ramps slowly in big swarms; allow
	// dialing up to the connection limit at once.
	cfg.HalfOpenConnsPerTorrent = settings.BTsets.ConnectionsLimit
	// Tracker/DHT announcers go idle once the known-peer pool passes
	// LowWater and resume only when it drains below it. Swarm address
	// lists are mostly stale, so the pool burns to zero within a minute
	// and then waits out the tracker interval with few connections.
	// High LowWater keeps peer discovery running continuously; addresses
	// are tiny, so a larger pool costs no meaningful RAM.
	cfg.TorrentPeersLowWater = 200
	cfg.TorrentPeersHighWater = 1000
	// The accept limiter masks inbound IPs to /24, so one connected peer
	// blocks every other peer from the same subnet for up to 15 minutes.
	// Fast seeders (seedboxes) cluster in shared provider subnets, which
	// is exactly the traffic this rejects; upstream later removed the
	// limiter entirely.
	cfg.DisableAcceptRateLimiting = true
	// Swarm addresses are mostly dead; a 20s dial timeout on each stale
	// address stalls the hunt for live peers when few candidates remain
	// (with a large pool the library already shrinks it towards the 3s
	// minimum).
	cfg.NominalDialTimeout = 10 * time.Second
}
