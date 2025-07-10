package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"driftdetector/domain/models"
	repositories "driftdetector/domain/repositories"
)

// StateParser defines the interface for parsing Terraform state files
type StateParser interface {
	ParseState(ctx context.Context, path string) (*models.TerraformState, error)
}

// StateFileParser implements StateParser for actual file system operations
type StateFileParser struct{}

// TerraformRepository implements the TerraformStateRepository interface
type TerraformRepository struct {
	parser StateParser
}

// NewTerraformRepository creates a new TerraformRepository with the given parser
func NewTerraformRepository(parser StateParser) repositories.TerraformStateRepository {
	if parser == nil {
		parser = &StateFileParser{}
	}
	return &TerraformRepository{
		parser: parser,
	}
}

// GetInstanceConfigs extracts instance configurations from a Terraform state file
func (r *TerraformRepository) GetInstanceConfigs(ctx context.Context, statePath string) ([]*models.Instance, error) {
	state, err := r.parser.ParseState(ctx, statePath)
	if err != nil {
		return nil, fmt.Errorf("parsing Terraform state: %w", err)
	}

	return r.extractInstances(state), nil
}

// GetInstanceConfigsFromDir extracts instance configurations from all Terraform state files in a directory
func (r *TerraformRepository) GetInstanceConfigsFromDir(ctx context.Context, dir string) ([]*models.Instance, error) {
	var instances []*models.Instance

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-json files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Try to parse the file as a Terraform state
		stateInstances, err := r.GetInstanceConfigs(ctx, path)
		if err != nil {
			// Skip files that don't contain valid Terraform state
			return nil
		}

		instances = append(instances, stateInstances...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}

	return instances, nil
}

// extractInstances converts Terraform state resources to domain models
func (r *TerraformRepository) extractInstances(state *models.TerraformState) []*models.Instance {
	var instances []*models.Instance

	// TODO: Implement actual extraction of instances from Terraform state
	// This is a placeholder implementation
	// You'll need to parse the state resources and convert them to your domain model

	return instances
}

// ParseState reads and parses a Terraform state file
func (p *StateFileParser) ParseState(ctx context.Context, path string) (*models.TerraformState, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state models.TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshaling Terraform state: %w", err)
	}

	return &state, nil
}
