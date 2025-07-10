package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"driftdetector/application"
	"driftdetector/domain/models"
)

// NewListDDDCmd creates a new list command using the DDD structure
func NewListDDDCmd() *cobra.Command {
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
			// Initialize application container
			container, err := application.NewContainer(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to initialize application container: %w", err)
			}

			tfRepo := container.GetTerraformRepository()
			ctx := cmd.Context()

			var instances []*models.Instance
			if tfState != "" {
				instances, err = tfRepo.GetInstanceConfigs(ctx, tfState)
			} else {
				instances, err = tfRepo.GetInstanceConfigsFromDir(ctx, tfDir)
			}

			if err != nil {
				return fmt.Errorf("failed to list instances from Terraform: %w", err)
			}

			// Display results
			if len(instances) == 0 {
				fmt.Println("No EC2 instance configurations found in the Terraform files.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "INSTANCE ID\tINSTANCE TYPE\tAMI\tTAGS")

			for _, instance := range instances {
				if instance == nil {
					continue
				}

				// Get tags as a string
				tagsStr := ""
				for k, v := range instance.Tags {
					tagsStr += fmt.Sprintf("%s=%s ", k, v)
				}

				// Use "-" for empty values to improve readability
				instanceID := instance.ID
				if instanceID == "" {
					instanceID = "-"
				}
				instanceType := instance.Type
				if instanceType == "" {
					instanceType = "-"
				}
				ami := instance.AMI
				ami = "-"
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
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&tfState, "tf-state", "s", "", "Path to Terraform state file")
	cmd.Flags().StringVarP(&tfDir, "tf-dir", "d", ".", "Path to Terraform configuration directory")

	// Mark flags as mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("tf-state", "tf-dir")

	return cmd
}
