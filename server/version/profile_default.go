//go:build !slim && !full

package version

// Profile is the build profile chosen by build tags: the default one is close to
// upstream TorrServer MatriX, -tags slim keeps only what a headless streaming
// server needs, -tags full adds everything the forks offer.
const Profile = ""
