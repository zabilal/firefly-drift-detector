package aws_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	awsrepo "driftdetector/infrastructure/aws"
)

// MockEC2API is a mock implementation of the EC2API interface
type MockEC2API struct {
	mock.Mock
}

func (m *MockEC2API) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

func (m *MockEC2API) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeVolumesOutput), args.Error(1)
}

func TestNewEC2Repository(t *testing.T) {
	// Given
	mockClient := new(MockEC2API)

	// When
	repo := awsrepo.NewEC2Repository(mockClient)

	// Then
	assert.NotNil(t, repo, "Repository should not be nil")
}

func TestEC2Repository_FindAll(t *testing.T) {
	// Given
	mockClient := new(MockEC2API)
	repo := awsrepo.NewEC2Repository(mockClient)

	t.Run("successful call", func(t *testing.T) {
		// Setup mock
		mockClient.On("DescribeInstances", mock.Anything, mock.Anything).Return(&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						{
							InstanceId: aws.String("i-1234567890abcdef0"),
							State:      &types.InstanceState{Name: "running"},
						},
					},
				},
			},
		}, nil)

		// When
		instances, err := repo.FindAll(context.Background())

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.Len(t, instances, 1, "Should return one instance")
		assert.Equal(t, "i-1234567890abcdef0", instances[0].ID, "Instance ID should match")
	})

	t.Run("error from API call", func(t *testing.T) {
		// Setup mock
		expectedErr := assert.AnError
		mockClient.On("DescribeInstances", mock.Anything, mock.Anything).Return((*ec2.DescribeInstancesOutput)(nil), expectedErr)

		// When
		instances, err := repo.FindAll(context.Background())

		// Then
		assert.ErrorIs(t, err, expectedErr, "Should return the expected error")
		assert.Nil(t, instances, "Should not return any instances on error")
	})
}

func TestEC2Repository_GetByID(t *testing.T) {
	// Given
	mockClient := new(MockEC2API)
	repo := awsrepo.NewEC2Repository(mockClient)
	instanceID := "i-1234567890abcdef0"

	t.Run("successful call", func(t *testing.T) {
		// Setup mock
		mockClient.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			if len(input.InstanceIds) != 1 {
				return false
			}
			return input.InstanceIds[0] == instanceID
		})).Return(&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						{
							InstanceId: aws.String(instanceID),
							State:      &types.InstanceState{Name: "running"},
						},
					},
				},
			},
		}, nil)

		// When
		instance, err := repo.GetByID(context.Background(), instanceID)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, instance, "Should return an instance")
		assert.Equal(t, instanceID, instance.ID, "Instance ID should match")
	})

	t.Run("instance not found", func(t *testing.T) {
		// Setup mock
		mockClient.On("DescribeInstances", mock.Anything, mock.Anything).Return(&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{},
		}, nil)

		// When
		instance, err := repo.GetByID(context.Background(), "nonexistent-instance")

		// Then
		assert.Error(t, err, "Should return an error")
		assert.Nil(t, instance, "Should not return an instance")
	})
}
