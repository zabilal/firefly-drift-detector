package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yourusername/driftdetector/internal/models"
	"github.com/yourusername/driftdetector/internal/terraform"
)

// NewListCmd creates a new list command
func NewListCmd() *cobra.Command {
	var (
		tfState string
		tfDir   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List EC2 instances managed by Terraform",
		Long: `List all EC2 instances that are managed by Terraform in the specified
state file or directory. This helps identify which instances can be checked for drift.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, tfState, tfDir, args)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&tfState, "tf-state", "s", "", "Path to Terraform state file")
	cmd.Flags().StringVarP(&tfDir, "tf-dir", "d", ".", "Path to Terraform configuration directory")

	// Mark flags as mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("tf-state", "tf-dir")

	return cmd
}

func runList(cmd *cobra.Command, tfState, tfDir string, args []string) error {
	// Validate inputs
	if tfState == "" && tfDir == "" {
		return fmt.Errorf("either --tf-state or --tf-dir must be specified")
	}

	// Parse Terraform configuration
	tfParser := terraform.NewParser()
	var configs []*models.InstanceConfig
	var err error

	if tfState != "" {
		configs, err = tfParser.ParseState(tfState)
	} else {
		// For directories, parse all .tf files
		entries, err := os.ReadDir(tfDir)
		if err != nil {
			return fmt.Errorf("failed to read Terraform directory: %v", err)
		}

		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".tf" {
				tfPath := filepath.Join(tfDir, entry.Name())
				config, err := tfParser.ParseHCL(tfPath)
				if err != nil {
					return fmt.Errorf("failed to parse Terraform file %s: %v", tfPath, err)
				}
				if config != nil {
					configs = append(configs, config)
				}
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to list instances from Terraform: %v", err)
	}

	// Display results
	if len(configs) == 0 {
		fmt.Println("No EC2 instance configurations found in the Terraform files.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INSTANCE ID\tINSTANCE TYPE\tAMI\tTAGS")

	for _, config := range configs {
		if config == nil {
			continue
		}

		// Get tags as a string
		tagsStr := ""
		for k, v := range config.Tags {
			tagsStr += fmt.Sprintf("%s=%s ", k, v)
		}

		// Use "-" for empty values to improve readability
		instanceID := config.InstanceID
		if instanceID == "" {
			instanceID = "-"
		}
		instanceType := config.InstanceType
		if instanceType == "" {
			instanceType = "-"
		}
		ami := config.AMI
		if ami == "" {
			ami = "-"
		}
		if tagsStr == "" {
			tagsStr = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", 
			instanceID,
			instanceType,
			ami,
			tagsStr,
		)
	}

	w.Flush()

	return nil
}
