//go:build slim && newui

package version

// The slim profile has no web UI, so -tags newui makes no sense with it: with both tags this file stops
// the build on the undefined name below.
var _ = build_tags_slim_and_newui_are_mutually_exclusive
