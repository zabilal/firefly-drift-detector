package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/yourusername/driftdetector/internal/logger"
	"github.com/yourusername/driftdetector/internal/models"
)

// Common AWS errors that we want to handle specifically
var (
	ErrInstanceNotFound = errors.New("instance not found")
	ErrInvalidRegion    = errors.New("invalid AWS region")
	ErrAccessDenied     = errors.New("access denied to AWS resources")
)

// EC2ClientWrapper wraps the AWS EC2 client to make it more testable
type EC2ClientWrapper struct {
	Client EC2DescribeInstancesAPI
	logger *logger.Logger
}

// EC2DescribeInstancesAPI defines the interface for the DescribeInstances API
// This interface allows us to mock the EC2 client in tests
type EC2DescribeInstancesAPI interface {
	DescribeInstances(
		ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeInstancesOutput, error)
}

// NewEC2ClientWrapper creates a new EC2 client wrapper with the default AWS configuration
func NewEC2ClientWrapper(ctx context.Context, region string) (*EC2ClientWrapper, error) {
	if region == "" {
		return nil, fmt.Errorf("%w: region cannot be empty", ErrInvalidRegion)
	}

	// Load the AWS configuration
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(so *retry.StandardOptions) {
				so.MaxAttempts = 5 // Maximum number of retry attempts
			})
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create a logger with context
	log := logger.WithFields(map[string]interface{}{
		"component": "aws-client",
		"region":    region,
	})

	log.Info("Initialized AWS EC2 client")

	return &EC2ClientWrapper{
		Client: ec2.NewFromConfig(cfg),
		logger: log,
	}, nil
}

// GetInstanceConfig retrieves the configuration of the specified EC2 instance
func (w *EC2ClientWrapper) GetInstanceConfig(ctx context.Context, instanceID string) (*models.InstanceConfig, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("%w: instance ID cannot be empty", ErrInstanceNotFound)
	}

	// Add instance ID to logger context
	log := w.logger.WithFields(map[string]interface{}{
		"instance_id": instanceID,
	})

	log.Debug("Fetching instance configuration")

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := w.Client.DescribeInstances(ctx, input)
	if err != nil {
			// Check for access denied errors
		var apiErr interface{
			Error() string
			ErrorCode() string
		}
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "UnauthorizedOperation":
				log.Error("Access denied when describing instance")
				return nil, fmt.Errorf("access denied: %v", err)
			case "InvalidInstanceID.NotFound":
				log.Warn("Instance not found")
				return nil, fmt.Errorf("no instance found with ID %s: %v", instanceID, err)
			case "InvalidInstanceID.Malformed":
				log.Warn("Invalid instance ID format")
				return nil, fmt.Errorf("no instance found with ID %s: invalid format", instanceID)
			}
		}

		log.Error("Failed to describe instance: %v", err)
		return nil, fmt.Errorf("failed to describe instance: %v", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		log.Warn("No instance found with the specified ID")
		return nil, fmt.Errorf("no instance found with ID %s", instanceID)
	}

	instance := result.Reservations[0].Instances[0]
	config := convertToInstanceConfig(instance)

	log.Info("Successfully retrieved instance configuration: instance_type=%s, state=%s",
		config.InstanceType,
		instance.State.Name,
	)

	return config, nil
}
