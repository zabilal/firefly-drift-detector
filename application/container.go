package application

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	detectionsvc "driftdetector/domain/services"
	awsrepo "driftdetector/infrastructure/aws"
	"driftdetector/infrastructure/terraform"
	tfrepo "driftdetector/infrastructure/terraform"
	repositories "driftdetector/domain/repositories"
)

// Container holds all the application dependencies
type Container struct {
	// Repositories
	instanceRepo repositories.InstanceRepository
	tfRepo      repositories.TerraformStateRepository

	// Services
	detectionSvc detectionsvc.DetectionService

	// Factories
	awsFactory awsrepo.ClientFactory
	tfParser   terraform.StateParser

	// AWS Config
	awsConfig aws.Config
}

// ContainerOption is a function that configures the container
type ContainerOption func(*Container) error

// WithAWSConfig allows setting a custom AWS config
func WithAWSConfig(cfg aws.Config) ContainerOption {
	return func(c *Container) error {
		c.awsConfig = cfg
		return nil
	}
}

// WithAWSFactory allows setting a custom AWS client factory
func WithAWSFactory(factory awsrepo.ClientFactory) ContainerOption {
	return func(c *Container) error {
		if factory == nil {
			return fmt.Errorf("AWS client factory cannot be nil")
		}
		c.awsFactory = factory
		return nil
	}
}

// WithTerraformParser allows setting a custom Terraform state parser
func WithTerraformParser(parser terraform.StateParser) ContainerOption {
	return func(c *Container) error {
		c.tfParser = parser
		return nil
	}
}

// NewContainer creates a new application container with all dependencies
func NewContainer(ctx context.Context, opts ...ContainerOption) (*Container, error) {
	// Create container with default values
	container := &Container{
		awsFactory: awsrepo.NewClientFactory(),
		tfParser:   &terraform.StateFileParser{},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(container); err != nil {
			return nil, fmt.Errorf("applying container option: %w", err)
		}
	}

	// Initialize AWS config if not provided
	if container.awsConfig.Region == "" {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
		container.awsConfig = cfg
	}

	// Initialize AWS clients
	ec2Client := container.awsFactory.NewEC2Client(container.awsConfig)

	// Initialize repositories
	container.instanceRepo = awsrepo.NewEC2Repository(ec2Client)
	container.tfRepo = tfrepo.NewTerraformRepository(container.tfParser)

	// Initialize services
	container.detectionSvc = detectionsvc.NewDetectionService()

	return container, nil
}

// GetInstanceRepository returns the instance repository
func (c *Container) GetInstanceRepository() repositories.InstanceRepository {
	return c.instanceRepo
}

// GetTerraformRepository returns the Terraform state repository
func (c *Container) GetTerraformRepository() repositories.TerraformStateRepository {
	return c.tfRepo
}

// GetDetectionService returns the detection service
func (c *Container) GetDetectionService() detectionsvc.DetectionService {
	return c.detectionSvc
}

// GetAWSConfig returns the AWS config
func (c *Container) GetAWSConfig() aws.Config {
	return c.awsConfig
}
