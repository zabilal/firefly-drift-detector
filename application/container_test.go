package application_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"driftdetector/application"
	"driftdetector/domain/models"
	"driftdetector/domain/services"
	awsrepo "driftdetector/infrastructure/aws"
)

// MockAWSFactory is a test implementation of the AWS ClientFactory interface
type MockAWSFactory struct {
	NewEC2ClientFunc func(cfg aws.Config) awsrepo.EC2API
}

func (m *MockAWSFactory) NewEC2Client(cfg aws.Config) awsrepo.EC2API {
	if m.NewEC2ClientFunc != nil {
		return m.NewEC2ClientFunc(cfg)
	}
	// Return a default mock EC2API that satisfies the interface
	return &MockEC2API{}
}

// MockTerraformParser is a test implementation of the StateParser interface
type MockTerraformParser struct {
	ParseStateFunc func(ctx context.Context, path string) (*models.TerraformState, error)
}

func (m *MockTerraformParser) ParseState(ctx context.Context, path string) (*models.TerraformState, error) {
	if m.ParseStateFunc != nil {
		return m.ParseStateFunc(ctx, path)
	}
	return &models.TerraformState{}, nil
}

// MockEC2API is a test implementation of the EC2API interface
type MockEC2API struct {
	FindAllFunc          func(ctx context.Context) ([]*models.Instance, error)
	GetByIDFunc          func(ctx context.Context, id string) (*models.Instance, error)
	DescribeInstancesFunc func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVolumesFunc   func(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
}

// Implement the EC2API interface methods
func (m *MockEC2API) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if m.DescribeInstancesFunc != nil {
		return m.DescribeInstancesFunc(ctx, params, optFns...)
	}
	// Return empty result by default
	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{},
	}, nil
}

func (m *MockEC2API) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	if m.DescribeVolumesFunc != nil {
		return m.DescribeVolumesFunc(ctx, params, optFns...)
	}
	// Return empty result by default
	return &ec2.DescribeVolumesOutput{
		Volumes: []types.Volume{},
	}, nil
}

// Helper methods for testing
func (m *MockEC2API) FindAll(ctx context.Context) ([]*models.Instance, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx)
	}
	// Default implementation that uses the mock's DescribeInstances
	output, err := m.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	var instances []*models.Instance
	for _, res := range output.Reservations {
		for _, instance := range res.Instances {
			instances = append(instances, &models.Instance{
				ID: *instance.InstanceId,
			})
		}
	}

	// If no instances were found, return a default one
	if len(instances) == 0 {
		instances = append(instances, &models.Instance{ID: "i-1234567890abcdef0"})
	}

	return instances, nil
}

func (m *MockEC2API) GetByID(ctx context.Context, id string) (*models.Instance, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	// Default implementation that uses the mock's DescribeInstances
	output, err := m.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return nil, err
	}

	for _, res := range output.Reservations {
		for range res.Instances {
			// Return the first instance found with the given ID
			return &models.Instance{ID: id}, nil
		}
	}

	return nil, nil
}

func TestNewContainer(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation with default options", func(t *testing.T) {
		// When
		container, err := application.NewContainer(ctx)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, container, "Should return a container")
	})

	t.Run("successful creation with custom AWS config", func(t *testing.T) {
		// Given
		customConfig := aws.Config{
			Region: "us-west-2",
		}

		// When
		container, err := application.NewContainer(ctx,
			application.WithAWSConfig(customConfig),
		)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, container, "Should return a container")
	})

	t.Run("successful creation with custom AWS factory", func(t *testing.T) {
		// Given
		mockEC2 := &MockEC2API{
			DescribeInstancesFunc: func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				return &ec2.DescribeInstancesOutput{
					Reservations: []types.Reservation{
						{
							Instances: []types.Instance{
								{
									InstanceId: aws.String("i-1234567890abcdef0"),
								},
							},
						},
					},
				}, nil
			},
		}

		factory := &MockAWSFactory{
			NewEC2ClientFunc: func(cfg aws.Config) awsrepo.EC2API {
				return mockEC2
			},
		}

		// When
		container, err := application.NewContainer(ctx,
			application.WithAWSFactory(factory),
		)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, container, "Should return a container")
	})

	t.Run("successful creation with custom Terraform parser", func(t *testing.T) {
		// Given
		parser := &MockTerraformParser{}

		// When
		container, err := application.NewContainer(ctx,
			application.WithTerraformParser(parser),
		)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, container, "Should return a container")
	})
}

func TestContainer_Getters(t *testing.T) {
	// Given
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func() *application.Container
		testFunc func(*testing.T, *application.Container)
	}{
		{
			name: "GetInstanceRepository",
			setup: func() *application.Container {
				container, _ := application.NewContainer(ctx)
				return container
			},
			testFunc: func(t *testing.T, c *application.Container) {
				repo := c.GetInstanceRepository()
				assert.NotNil(t, repo, "Should return an instance repository")
			},
		},
		{
			name: "GetTerraformRepository",
			setup: func() *application.Container {
				container, _ := application.NewContainer(ctx)
				return container
			},
			testFunc: func(t *testing.T, c *application.Container) {
				repo := c.GetTerraformRepository()
				assert.NotNil(t, repo, "Should return a Terraform repository")
			},
		},
		{
			name: "GetDetectionService",
			setup: func() *application.Container {
				container, _ := application.NewContainer(ctx)
				return container
			},
			testFunc: func(t *testing.T, c *application.Container) {
				svc := c.GetDetectionService()
				assert.NotNil(t, svc, "Should return a detection service")
			},
		},
		{
			name: "GetAWSConfig",
			setup: func() *application.Container {
				container, _ := application.NewContainer(ctx)
				return container
			},
			testFunc: func(t *testing.T, c *application.Container) {
				cfg := c.GetAWSConfig()
				assert.NotNil(t, cfg, "Should return an AWS config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.setup()
			tt.testFunc(t, container)
		})
	}
}

// MockDetectionService is a test implementation of the DetectionService interface
type MockDetectionService struct {
	services.DetectionService
}

func (m *MockDetectionService) DetectDrift(actual, desired *models.Instance) (*models.DriftReport, error) {
	return &models.DriftReport{
		InstanceID: actual.ID,
		Drifts:     []models.Drift{},
	}, nil
}
