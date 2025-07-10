package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	tfjson "github.com/hashicorp/terraform-json"
	"driftdetector/domain/models"
	"driftdetector/domain/repositories"
)

// Ensure TerraformStateRepository implements the TerraformStateRepository interface
var _ repositories.TerraformStateRepository = (*TerraformStateRepository)(nil)

// TerraformStateRepository implements the TerraformStateRepository interface
type TerraformStateRepository struct {
	// Add any dependencies here (e.g., logger, config)
}

// NewTerraformStateRepository creates a new TerraformStateRepository
func NewTerraformStateRepository() *TerraformStateRepository {
	return &TerraformStateRepository{}
}

// GetInstanceConfigs extracts instance configurations from a Terraform state file
func (r *TerraformStateRepository) GetInstanceConfigs(ctx context.Context, statePath string) ([]*models.Instance, error) {
	// Read the state file
	stateData, err := ioutil.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse the state file
	var state tfjson.State
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Extract instance configurations
	return r.extractInstancesFromState(&state)
}

// GetInstanceConfigsFromDir extracts instance configurations from a Terraform directory
func (r *TerraformStateRepository) GetInstanceConfigsFromDir(ctx context.Context, dir string) ([]*models.Instance, error) {
	// Look for terraform.tfstate or terraform.tfstate.d directory
	statePath := filepath.Join(dir, "terraform.tfstate")
	if _, err := os.Stat(statePath); err == nil {
		return r.GetInstanceConfigs(ctx, statePath)
	}

	// Check for terraform.tfstate.d directory
	tfStateDir := filepath.Join(dir, "terraform.tfstate.d")
	if _, err := os.Stat(tfStateDir); err == nil {
		// Get the most recent state file
		files, err := ioutil.ReadDir(tfStateDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read terraform.tfstate.d directory: %w", err)
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("no state files found in terraform.tfstate.d")
		}

		// Sort by modification time (newest first)
		var latestFile os.FileInfo
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if latestFile == nil || file.ModTime().After(latestFile.ModTime()) {
				latestFile = file
			}
		}

		if latestFile == nil {
			return nil, fmt.Errorf("no valid state files found in terraform.tfstate.d")
		}

		return r.GetInstanceConfigs(ctx, filepath.Join(tfStateDir, latestFile.Name()))
	}

	return nil, fmt.Errorf("no terraform state file found in %s", dir)
}

// extractInstancesFromState extracts instance configurations from a parsed Terraform state
func (r *TerraformStateRepository) extractInstancesFromState(state *tfjson.State) ([]*models.Instance, error) {
	var instances []*models.Instance

	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		return instances, nil
	}

	// Process all resources in the root module
	instances = append(instances, r.extractInstancesFromModule(state.Values.RootModule)...)

	// Process child modules if they exist in the root module's ModuleCalls
	if state.Values.RootModule.ChildModules != nil {
		for _, module := range state.Values.RootModule.ChildModules {
			instances = append(instances, r.extractInstancesFromModule(module)...)
		}
	}

	return instances, nil
}

// extractInstancesFromModule extracts instance configurations from a Terraform module
func (r *TerraformStateRepository) extractInstancesFromModule(module *tfjson.StateModule) []*models.Instance {
	var instances []*models.Instance

	if module == nil {
		return instances
	}

	for _, resource := range module.Resources {
		if resource.Type != "aws_instance" {
			continue
		}

		// Parse the instance attributes
		instance, err := r.parseInstanceResource(resource)
		if err != nil {
			// Log the error but continue with other instances
			continue
		}

		instances = append(instances, instance)
	}

	return instances
}

// parseInstanceResource parses a Terraform resource into an Instance
func (r *TerraformStateRepository) parseInstanceResource(resource *tfjson.StateResource) (*models.Instance, error) {
	if resource == nil || resource.AttributeValues == nil {
		return nil, fmt.Errorf("invalid resource")
	}

	// Extract basic instance information
	attrs := resource.AttributeValues
	instanceID, _ := attrs["id"].(string)
	instanceType, _ := attrs["instance_type"].(string)
	ami, _ := attrs["ami"].(string)

	// Create a new instance with required fields
	instance := models.NewInstance(instanceID, instanceType, ami)

	// Extract optional fields
	if v, ok := attrs["key_name"].(string); ok {
		instance.KeyName = v
	}

	if v, ok := attrs["subnet_id"].(string); ok {
		instance.SubnetID = v
	}

	if v, ok := attrs["vpc_id"].(string); ok {
		instance.VPCID = v
	}

	if v, ok := attrs["private_ip"].(string); ok {
		instance.PrivateIPAddress = v
	}

	if v, ok := attrs["public_ip"].(string); ok {
		instance.PublicIPAddress = v
	}

	if v, ok := attrs["private_dns"].(string); ok {
		instance.PrivateDNSName = v
	}

	if v, ok := attrs["public_dns"].(string); ok {
		instance.PublicDNSName = v
	}

	// Extract tags
	if tags, ok := attrs["tags"].(map[string]interface{}); ok {
		for k, v := range tags {
			if strVal, ok := v.(string); ok {
				instance.AddTag(k, strVal)
			}
		}
	}

	// Extract security groups
	if sgs, ok := attrs["vpc_security_group_ids"].([]interface{}); ok {
		for _, sg := range sgs {
			if sgID, ok := sg.(string); ok {
				instance.SecurityGroups = append(instance.SecurityGroups, models.SecurityGroup{
					GroupID: sgID,
				})
			}
		}
	}

	// Extract root block device configuration
	if rootBlockDevice, ok := attrs["root_block_device"].([]interface{}); ok && len(rootBlockDevice) > 0 {
		if rootDevice, ok := rootBlockDevice[0].(map[string]interface{}); ok {
			if size, ok := rootDevice["volume_size"].(float64); ok {
				instance.RootVolumeSize = int(size)
			}

			if volType, ok := rootDevice["volume_type"].(string); ok {
				instance.RootVolumeType = volType
			}

			if iops, ok := rootDevice["iops"].(float64); ok {
				instance.RootVolumeIops = int(iops)
			}

			if encrypted, ok := rootDevice["encrypted"].(bool); ok {
				encryptedVal := encrypted
				instance.RootVolumeEncrypted = &encryptedVal
			}
		}
	}

	// Extract monitoring configuration
	if monitoring, ok := attrs["monitoring"].(bool); ok {
		monitoringVal := monitoring
		instance.Monitoring = &monitoringVal
	}

	// Extract IAM instance profile
	if iamProfile, ok := attrs["iam_instance_profile"].(string); ok {
		instance.IAMInstanceProfile = iamProfile
	}

	return instance, nil
}
