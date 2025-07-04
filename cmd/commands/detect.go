package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/driftdetector/internal/aws"
	"github.com/yourusername/driftdetector/internal/detector"
	"github.com/yourusername/driftdetector/internal/logger"
	"github.com/yourusername/driftdetector/internal/models"
	"github.com/yourusername/driftdetector/internal/terraform"
)

// NewDetectCmd creates a new detect command
func NewDetectCmd() *cobra.Command {
	var (
		instanceID string
		tfState    string
		tfDir      string
	)

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect configuration drift for an EC2 instance",
		Long: `Detect configuration drift for an EC2 instance by comparing its current
AWS configuration with the Terraform state or configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetect(cmd, instanceID, tfState, tfDir, args)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&instanceID, "instance", "i", "", "EC2 instance ID to check for drift")
	cmd.Flags().StringVarP(&tfState, "tf-state", "s", "", "Path to Terraform state file")
	cmd.Flags().StringVarP(&tfDir, "tf-dir", "d", ".", "Path to Terraform configuration directory")

	// Mark required flags
	cmd.MarkFlagRequired("instance")

	// Mark flags as mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("tf-state", "tf-dir")

	return cmd
}

// findMatchingConfig finds the Terraform configuration that matches the given AWS instance
func findMatchingConfig(awsConfig *models.InstanceConfig, tfConfigs []*models.InstanceConfig, log *logger.Logger) *models.InstanceConfig {
	// First try to match by instance ID
	for _, cfg := range tfConfigs {
		if cfg.InstanceID == awsConfig.InstanceID {
			log.Debug("Found matching config by instance ID", "instance_id", cfg.InstanceID)
			return cfg
		}
	}

	// Then try to match by tags (Name tag is most common)
	if nameTag, hasName := awsConfig.Tags["Name"]; hasName {
		for _, cfg := range tfConfigs {
			if cfgName, exists := cfg.Tags["Name"]; exists && cfgName == nameTag {
				log.Debug("Found matching config by Name tag", "name", nameTag)
				return cfg
			}
		}
	}

	log.Warn("No exact matching Terraform configuration found, using first available config")
	if len(tfConfigs) > 0 {
		return tfConfigs[0]
	}
	return nil
}

func runDetect(cmd *cobra.Command, instanceID, tfState, tfDir string, args []string) error {
	// Initialize logger
	log := logger.NewLogger(logger.Config{
		Level:  logger.LevelInfo,
		Output: os.Stdout,
	})

	// Validate inputs
	if instanceID == "" {
		return fmt.Errorf("instance ID is required")
	}

	if tfState == "" && tfDir == "" {
		return fmt.Errorf("either --tf-state or --tf-dir must be specified")
	}

	// Initialize AWS client
	log.Info("Initializing AWS client...")
	awsClient, err := aws.NewEC2Client(context.Background(), awsRegion)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %v", err)
	}

	// Get current instance configuration from AWS
	log.Info("Fetching instance configuration from AWS...", "instance_id", instanceID)
	awsConfig, err := awsClient.GetInstanceConfig(cmd.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance config from AWS: %v", err)
	}

	// Parse Terraform configuration
	log.Info("Parsing Terraform configuration...")
	var tfConfigs []*models.InstanceConfig
	tfParser := terraform.NewParser()

	if tfState != "" {
		log.Debug("Parsing Terraform state file", "path", tfState)
		tfConfigs, err = tfParser.ParseState(tfState)
		if err != nil {
			return fmt.Errorf("failed to parse Terraform state: %v", err)
		}
	} else if tfDir != "" {
		log.Debug("Scanning Terraform directory", "path", tfDir)
		// For directory, look for .tf files and parse them
		entries, err := os.ReadDir(tfDir)
		if err != nil {
			return fmt.Errorf("failed to read Terraform directory: %v", err)
		}

		foundConfigs := false
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".tf" {
				tfPath := filepath.Join(tfDir, entry.Name())
				log.Debug("Parsing Terraform file", "file", tfPath)
				tfConfig, err := tfParser.ParseHCL(tfPath)
				if err != nil {
					log.Error("Failed to parse Terraform file", "file", tfPath, "error", err)
					continue
				}
				if tfConfig != nil {
					tfConfigs = append(tfConfigs, tfConfig)
					foundConfigs = true
				}
			}
		}

		if !foundConfigs {
			return fmt.Errorf("no valid Terraform configurations found in directory: %s", tfDir)
		}
	}

	if len(tfConfigs) == 0 {
		return fmt.Errorf("no Terraform configurations found")
	}

	// Find the matching Terraform configuration
	log.Info("Matching AWS instance with Terraform configuration...")
	tfConfig := findMatchingConfig(awsConfig, tfConfigs, log)
	if tfConfig == nil {
		return fmt.Errorf("no matching Terraform configuration found for instance %s", instanceID)
	}

	// Initialize the detector
	d := detector.NewDetector()

	// Detect drift
	log.Info("Detecting configuration drift...")
	report := d.DetectDrift(awsConfig, tfConfig)

	// Display results
	log.Info("\n=== Drift Detection Report ===")
	log.Info("%s", fmt.Sprintf("Instance ID: %s", awsConfig.InstanceID))
	log.Info("%s", fmt.Sprintf("Has Drift: %v", report.HasDrift))

	if report.HasDrift {
		log.Warn("\nDrift detected!")
		for _, drift := range report.Drifts {
			switch drift.Type {
			case detector.DriftTypeAdded:
				log.Warn("%s", fmt.Sprintf("[ADDED] %s: %v (not in Terraform)", drift.Path, drift.Actual))
			case detector.DriftTypeRemoved:
				log.Warn("%s", fmt.Sprintf("[REMOVED] %s: %v (in Terraform but not in AWS)", drift.Path, drift.Expected))
			case detector.DriftTypeModified:
				log.Warn("%s", fmt.Sprintf("[MODIFIED] %s: AWS=%v, Terraform=%v", drift.Path, drift.Actual, drift.Expected))
			}
		}
	} else {
		log.Info("\nNo drift detected. The instance matches its Terraform configuration.")
	}

	return nil
}

