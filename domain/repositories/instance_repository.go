package repositories

import (
	"context"
	"driftdetector/domain/models"
)

// InstanceRepository defines the interface for instance persistence operations
type InstanceRepository interface {
	// GetByID retrieves an instance by its ID
	GetByID(ctx context.Context, id string) (*models.Instance, error)
	
	// GetByIDs retrieves multiple instances by their IDs
	GetByIDs(ctx context.Context, ids []string) ([]*models.Instance, error)
	
	// FindAll retrieves all instances (with pagination support if needed)
	FindAll(ctx context.Context) ([]*models.Instance, error)
	
	// Save persists an instance
	Save(ctx context.Context, instance *models.Instance) error
	
	// Delete removes an instance
	Delete(ctx context.Context, id string) error
}

// DriftDetectionRepository defines the interface for drift detection operations
type DriftDetectionRepository interface {
	// DetectDrift compares actual and desired instance states
	DetectDrift(actual, desired *models.Instance) (*models.DriftReport, error)
	
	// GetDriftHistory retrieves historical drift reports for an instance
	GetDriftHistory(instanceID string, limit int) ([]*models.DriftReport, error)
}

// TerraformStateRepository defines the interface for accessing Terraform state
type TerraformStateRepository interface {
	// GetInstanceConfigs extracts instance configurations from Terraform state
	GetInstanceConfigs(ctx context.Context, statePath string) ([]*models.Instance, error)
	
	// GetInstanceConfigsFromDir extracts instance configurations from Terraform directory
	GetInstanceConfigsFromDir(ctx context.Context, dir string) ([]*models.Instance, error)
}
