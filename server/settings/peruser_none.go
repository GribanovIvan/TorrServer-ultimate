//go:build !full

package settings

import "net/http"

// Viewed marks are shared by all accounts: per-user marks are in the full
// profile (peruser.go).

func PerUserStartup() {}

func ViewedKeyForRequest(hash string, _ *http.Request) string { return hash }

func SetViewedForUser(vv *Viewed, _ string) { SetViewed(vv) }

func RemViewedForUser(vv *Viewed, _ string) { RemViewed(vv) }

func ListViewedForUser(hash, _ string) []*Viewed { return ListViewed(hash) }
