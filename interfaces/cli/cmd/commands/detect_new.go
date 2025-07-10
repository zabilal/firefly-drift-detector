package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"driftdetector/application"
	"driftdetector/domain/models"
)

// NewDetectDDDCmd creates a new detect command with the new DDD structure
func NewDetectDDDCmd() *cobra.Command {
	var (
		instanceID    string
		stateFile     string
		tfDir         string
		outputFormat  string
		showAll       bool
		showOnlyDrift bool
	)

	cmd := &cobra.Command{
		Use:   "detect-ddd",
		Short: "Detect configuration drift using DDD structure",
		Long: `Detect configuration drift between AWS EC2 instances and their Terraform configuration
using the new Domain-Driven Design structure.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize application container
			container, err := application.NewContainer(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to initialize application container: %w", err)
			}

			detectionSvc := container.GetDetectionService()

			// Get the instance from AWS
			instance, err := container.GetInstanceRepository().GetByID(cmd.Context(), instanceID)
			if err != nil {
				return fmt.Errorf("failed to fetch instance from AWS: %w", err)
			}

			// Get desired state from Terraform
			var instances []*models.Instance
			if stateFile != "" {
				instances, err = container.GetTerraformRepository().GetInstanceConfigs(cmd.Context(), stateFile)
			} else if tfDir != "" {
				instances, err = container.GetTerraformRepository().GetInstanceConfigsFromDir(cmd.Context(), tfDir)
			} else {
				return fmt.Errorf("either --state-file or --tf-dir must be specified")
			}

			if err != nil {
				return fmt.Errorf("failed to get desired state from Terraform: %w", err)
			}

			// Find the specific instance in the results
			var desiredInstance *models.Instance
			for _, inst := range instances {
				if inst.ID == instanceID {
					desiredInstance = inst
					break
				}
			}

			if desiredInstance == nil {
				return fmt.Errorf("instance %s not found in Terraform state", instanceID)
			}

			// Detect drift
			report, err := detectionSvc.DetectDrift(cmd.Context(), instance, desiredInstance)
			if err != nil {
				return fmt.Errorf("failed to detect drift: %w", err)
			}

			// Output results
			return outputResults(report, outputFormat, showAll, showOnlyDrift)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&instanceID, "instance", "i", "", "EC2 instance ID to check for drift (required)")
	cmd.Flags().StringVarP(&stateFile, "state-file", "s", "", "Path to Terraform state file")
	cmd.Flags().StringVarP(&tfDir, "tf-dir", "d", "", "Path to Terraform configuration directory")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json)")
	cmd.Flags().BoolVar(&showAll, "all", false, "Show all fields, even those without drift")
	cmd.Flags().BoolVar(&showOnlyDrift, "only-drift", false, "Show only fields with drift")

	// Mark required flags
	if err := cmd.MarkFlagRequired("instance"); err != nil {
		return nil
	}

	// Mark mutually exclusive flags
	cmd.MarkFlagsOneRequired("state-file", "tf-dir")
	cmd.MarkFlagsMutuallyExclusive("state-file", "tf-dir")

	return cmd
}

// outputResults prints the drift report in the specified format
func outputResults(report *models.DriftReport, format string, showAll, showOnlyDrift bool) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	case "text":
		return printTextReport(report, showAll, showOnlyDrift)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// printTextReport prints the drift report in a human-readable text format
func printTextReport(report *models.DriftReport, showAll, showOnlyDrift bool) error {
	fmt.Printf("Drift Report for Instance: %s\n", report.InstanceID)
	fmt.Printf("Drift Detected: %v\n", report.HasDrifts())
	fmt.Println(strings.Repeat("-", 80))

	if len(report.Drifts) == 0 {
		fmt.Println("No configuration drift detected.")
		return nil
	}

	for _, d := range report.Drifts {
		// Skip if we're only showing drifts and this isn't one
		if showOnlyDrift && d.Type == "" {
			continue
		}

		// Skip if we're not showing all fields and this isn't a drift
		if !showAll && d.Type == "" {
			continue
		}

		// Print drift details
		fmt.Printf("Path: %s\n", d.Path)
		if d.Type != "" {
			fmt.Printf("Type: %s\n", d.Type)
		}

		// Print expected/actual values if available
		if d.Expected != nil {
			fmt.Printf("Expected: %v\n", d.Expected)
		}
		if d.Actual != nil {
			fmt.Printf("Actual:   %v\n", d.Actual)
		}
		if d.Description != "" {
			fmt.Printf("Details:  %s\n", d.Description)
		}
		fmt.Println(strings.Repeat("-", 40))
	}

	return nil
}
