//go:build !full

package settings

// FullSets is empty outside the full profile (btsets_full.go), so the settings
// JSON has no fields of features that are not built in.
type FullSets struct{}
