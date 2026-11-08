//go:build full

package settings

// FullSets are the settings of features only the full profile has.
type FullSets struct {
	PerUserData bool // per-user data separation (freetushkan), see peruser.go
}
