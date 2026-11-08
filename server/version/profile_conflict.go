//go:build slim && full

package version

// The slim and full profiles exclude each other: with both tags this file stops
// the build on the undefined name below.
var _ = build_tags_slim_and_full_are_mutually_exclusive
