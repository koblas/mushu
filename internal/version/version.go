package version

import (
	"fmt"
	"runtime"
)

// These variables are set during build time using -ldflags
var (
	// Version is the semantic version of the application
	Version = "dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// BuildTime is the build timestamp
	BuildTime = "unknown"

	// GoVersion is the Go version used to build the application
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// String implements the fmt.Stringer interface
func (i Info) String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, go: %s)",
		i.Version, i.Commit, i.BuildTime, i.GoVersion)
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	}
}

// Short returns a short version string
func Short() string {
	return Version
}
