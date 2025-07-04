package cmd

import (
	"context"
	"encoding/json"
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
		mockMode   bool
		mockFile   string
	)

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect configuration drift for an EC2 instance",
		Long: `Detect configuration drift for an EC2 instance by comparing its current
AWS configuration with the Terraform state or configuration.

This command can run in two modes:
1. Live mode (default): Connects to AWS to fetch current instance configuration
2. Mock mode (--mock): Uses a local JSON file for testing without AWS credentials`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mockMode && mockFile == "" {
				return fmt.Errorf("--mock-file is required when --mock is enabled")
			}
			return runDetect(cmd, instanceID, tfState, tfDir, mockMode, mockFile, args)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&instanceID, "instance", "i", "", "EC2 instance ID to check for drift")
	cmd.Flags().StringVarP(&tfState, "tf-state", "s", "", "Path to Terraform state file")
	cmd.Flags().StringVarP(&tfDir, "tf-dir", "d", ".", "Path to Terraform configuration directory")
	cmd.Flags().BoolVar(&mockMode, "mock", false, "Enable mock mode (uses local JSON file instead of AWS)")
	cmd.Flags().StringVar(&mockFile, "mock-file", "", "Path to JSON file containing mock EC2 instance data (required for mock mode)")

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

// validateMockConfig validates the structure of the mock configuration
func validateMockConfig(config *models.InstanceConfig) error {
	if config.InstanceID == "" {
		return fmt.Errorf("mock configuration must include InstanceID")
	}

	// Validate tags if present
	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}

	// Validate security groups if present
	for i, sg := range config.SecurityGroups {
		if sg.GroupID == "" {
			return fmt.Errorf("security group at index %d is missing GroupID", i)
		}
	}

	return nil
}

// loadMockConfig loads and validates instance configuration from a local JSON file
func loadMockConfig(filePath, instanceID string) (*models.InstanceConfig, error) {
	// Check if file exists and is readable
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access mock file: %v", err)
	}

	// Check if it's a regular file
	if info.IsDir() {
		return nil, fmt.Errorf("mock file path is a directory, expected a file")
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock file: %v", err)
	}

	// Parse the JSON into our model
	var config models.InstanceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mock configuration: %v", err)
	}

	// Override instance ID if needed
	if instanceID != "" {
		config.InstanceID = instanceID
	}

	// Validate the configuration
	if err := validateMockConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid mock configuration: %v", err)
	}

	return &config, nil
}

func runDetect(cmd *cobra.Command, instanceID, tfState, tfDir string, mockMode bool, mockFile string, args []string) error {
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

	var awsConfig *models.InstanceConfig
	var err error

	if mockMode {
		// Mock mode: Load configuration from file
		log.Info("Running in mock mode, loading instance configuration from file...", "file", mockFile)
		awsConfig, err = loadMockConfig(mockFile, instanceID)
		if err != nil {
			return fmt.Errorf("failed to load mock configuration: %v", err)
		}
	} else {
		// Live mode: Connect to AWS
		log.Info("Initializing AWS client...")
		awsClient, err := aws.NewEC2Client(context.Background(), awsRegion)
		if err != nil {
			return fmt.Errorf("failed to create AWS client: %v\n\nNote: To run without AWS credentials, use --mock with --mock-file", err)
		}

		// Get current instance configuration from AWS
		log.Info("Fetching instance configuration from AWS...", "instance_id", instanceID)
		awsConfig, err = awsClient.GetInstanceConfig(cmd.Context(), instanceID)
		if err != nil {
			return fmt.Errorf("failed to get instance config from AWS: %v\n\nNote: To run without AWS credentials, use --mock with --mock-file", err)
		}
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

