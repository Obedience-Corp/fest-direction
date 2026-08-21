// Package version holds build metadata injected at link time via -ldflags
// (see justfile: version_pkg / ldflags).
package version

// Set by the build; defaults describe an untagged local build.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String renders the one-line form used by --version.
func String() string {
	return Version + " (" + Commit + ", built " + BuildDate + ")"
}
