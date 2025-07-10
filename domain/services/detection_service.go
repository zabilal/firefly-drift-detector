package services

import (
	"context"
	"driftdetector/domain/models"
)

// DetectionService defines the interface for drift detection operations
type DetectionService interface {
	// DetectDrift compares actual and desired instance states and returns a drift report
	DetectDrift(ctx context.Context, actual, desired *models.Instance) (*models.DriftReport, error)
	
	// BatchDetectDrift performs drift detection for multiple instances
	BatchDetectDrift(ctx context.Context, actual, desired []*models.Instance) (map[string]*models.DriftReport, error)
	
	// GetDriftHistory retrieves historical drift reports for an instance
	GetDriftHistory(instanceID string, limit int) ([]*models.DriftReport, error)
}

// DefaultDetectionService is the default implementation of DetectionService
type DefaultDetectionService struct {
	detector *DriftDetector
}

// NewDetectionService creates a new instance of DefaultDetectionService
func NewDetectionService() *DefaultDetectionService {
	return &DefaultDetectionService{
		detector: NewDriftDetector(),
	}
}

// DetectDrift implements the DetectionService interface
func (s *DefaultDetectionService) DetectDrift(ctx context.Context, actual, desired *models.Instance) (*models.DriftReport, error) {
	if actual == nil || desired == nil {
		return nil, ErrInvalidInput
	}

	if actual.ID != desired.ID {
		return nil, ErrInstanceMismatch
	}

	report := s.detector.CompareInstances(actual, desired)
	return report, nil
}

// BatchDetectDrift implements the DetectionService interface
func (s *DefaultDetectionService) BatchDetectDrift(
	ctx context.Context,
	actual, desired []*models.Instance,
) (map[string]*models.DriftReport, error) {
	reports := make(map[string]*models.DriftReport)

	// Create a map of desired instances by ID for quick lookup
	desiredMap := make(map[string]*models.Instance)
	for _, inst := range desired {
		desiredMap[inst.ID] = inst
	}

	// Compare each actual instance with its desired state
	for _, actualInst := range actual {
		if desiredInst, exists := desiredMap[actualInst.ID]; exists {
			report, err := s.DetectDrift(ctx, actualInst, desiredInst)
			if err != nil {
				return nil, err
			}
			reports[actualInst.ID] = report
		} else {
			// Handle case where instance exists in actual but not in desired
			report := models.NewDriftReport(actualInst.ID)
			report.AddDrift(models.NewDrift(
				models.DriftTypeRemoved,
				"",
				nil,
				nil,
				"Instance exists in actual state but not in desired state",
			))
			reports[actualInst.ID] = report
		}
	}

	// Check for instances that exist in desired but not in actual
	for _, desiredInst := range desired {
		if _, exists := reports[desiredInst.ID]; !exists {
			report := models.NewDriftReport(desiredInst.ID)
			report.AddDrift(models.NewDrift(
				models.DriftTypeAdded,
				"",
				nil,
				desiredInst,
				"Instance exists in desired state but not in actual state",
			))
			reports[desiredInst.ID] = report
		}
	}

	return reports, nil
}

// GetDriftHistory implements the DetectionService interface
func (s *DefaultDetectionService) GetDriftHistory(instanceID string, limit int) ([]*models.DriftReport, error) {
	// Implementation would typically query a persistence layer
	// This is a placeholder that returns an empty slice
	return []*models.DriftReport{}, nil
}

// Common errors
var (
	ErrInvalidInput     = NewDomainError("invalid input parameters")
	ErrInstanceMismatch = NewDomainError("instance IDs do not match")
)

// DomainError represents a domain-specific error
type DomainError struct {
	msg string
}

// NewDomainError creates a new DomainError
func NewDomainError(msg string) error {
	return &DomainError{msg: msg}
}

// Error implements the error interface
func (e *DomainError) Error() string {
	return e.msg
}
