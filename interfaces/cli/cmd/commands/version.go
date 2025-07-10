package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are set during build time using ldflags
var (
	// Version is the semantic version of the application
	Version = "dev"
	// Commit is the git commit hash
	Commit = "none"
	// Date is the build date
	Date = "unknown"
	// GoVersion is the Go version used to build the application
	GoVersion = "unknown"
)

// NewVersionCmd creates a new version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version information for driftdetector.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(GetVersion())
		},
	}
}


// GetVersion returns a formatted version string
func GetVersion() string {
	return fmt.Sprintf("driftdetector version %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, GoVersion)
}
