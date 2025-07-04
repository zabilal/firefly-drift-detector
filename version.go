package main

import "fmt"

// These variables are set during build time using ldflags
var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	GoVersion = "unknown"
)

// GetVersion returns a formatted version string
func GetVersion() string {
	return fmt.Sprintf("driftdetector version %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, GoVersion)
}
