package main

import (
	"fmt"
	"os"

	cmd "github.com/yourusername/driftdetector/cmd/commands"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
