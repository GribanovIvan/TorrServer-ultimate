//go:build !full

package settings

// DefaultConnectionsLimit is the default of BTSets.ConnectionsLimit; the full
// profile raises it to 40 (defaults_full.go, from Tommixoft/TorrServer).
const DefaultConnectionsLimit = 25
