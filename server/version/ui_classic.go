//go:build !newui

package version

// UI is the embedded web UI chosen by build tags: the classic MatriX UI by default, the NextGen one
// with -tags newui (shown in Full() as "-newui"). The slim profile has no web UI at all.
const UI = ""
