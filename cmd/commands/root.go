package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flags
var (
	awsRegion string
	outputFmt string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "driftdetector",
	Short: "Detect configuration drift between AWS EC2 instances and Terraform",
	Long: `DriftDetector is a tool that helps you identify configuration drift
between your AWS EC2 instances and their corresponding Terraform configurations.

It can detect changes in instance types, security groups, tags, and other
configuration parameters that might have been modified outside of your
infrastructure as code.`,
}

// NewRootCmd creates a new root command
func NewRootCmd() *cobra.Command {
	// Add version flag
	rootCmd.Version = Version
	
	// Add commands
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewDetectCmd())
	rootCmd.AddCommand(NewVersionCmd())
	
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&awsRegion, "region", "r", "", "AWS region (defaults to AWS_REGION environment variable)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "Output format (text, json)")
}
