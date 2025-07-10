package commands

import (
	"context"
	"fmt"

	"driftdetector/domain/models"
	"driftdetector/domain/repositories"
	"driftdetector/domain/services"
)

// DetectDriftCommand represents the command to detect drift for an instance
type DetectDriftCommand struct {
	InstanceID string
	TerraformStateFile string
	TerraformDir      string
}

// DetectDriftHandler handles the DetectDriftCommand
type DetectDriftHandler struct {
	detectionService services.DetectionService
	instanceRepo    repositories.InstanceRepository
	tfStateRepo     repositories.TerraformStateRepository
}

// NewDetectDriftHandler creates a new DetectDriftHandler
func NewDetectDriftHandler(
	detectionService services.DetectionService,
	instanceRepo repositories.InstanceRepository,
	tfStateRepo repositories.TerraformStateRepository,
) *DetectDriftHandler {
	return &DetectDriftHandler{
		detectionService: detectionService,
		instanceRepo:    instanceRepo,
		tfStateRepo:     tfStateRepo,
	}
}

// Handle processes the DetectDriftCommand
func (h *DetectDriftHandler) Handle(ctx context.Context, cmd DetectDriftCommand) (*models.DriftReport, error) {
	// Get actual instance from AWS
	actualInstance, err := h.instanceRepo.GetByID(ctx, cmd.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance from AWS: %w", err)
	}

	// Get desired state from Terraform
	var desiredInstances []*models.Instance
	if cmd.TerraformStateFile != "" {
		desiredInstances, err = h.tfStateRepo.GetInstanceConfigs(ctx, cmd.TerraformStateFile)
	} else if cmd.TerraformDir != "" {
		desiredInstances, err = h.tfStateRepo.GetInstanceConfigsFromDir(ctx, cmd.TerraformDir)
	} else {
		return nil, fmt.Errorf("either terraform state file or directory must be provided")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get desired state from Terraform: %w", err)
	}

	// Find the matching desired instance
	var desiredInstance *models.Instance
	for _, inst := range desiredInstances {
		if inst.ID == cmd.InstanceID {
			desiredInstance = inst
			break
		}
	}

	if desiredInstance == nil {
		return nil, fmt.Errorf("instance %s not found in Terraform state", cmd.InstanceID)
	}

	// Perform drift detection
	report, err := h.detectionService.DetectDrift(ctx, actualInstance, desiredInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to detect drift: %w", err)
	}

	return report, nil
}
