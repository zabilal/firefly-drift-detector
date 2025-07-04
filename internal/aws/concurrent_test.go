package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yourusername/driftdetector/internal/aws/testutils"
)

// TestGetInstanceConfigs_Success tests the successful retrieval of multiple instance configurations
func TestGetInstanceConfigs_Success(t *testing.T) {
	// Setup mock
	mockAPI := &testutils.MockEC2API{}
	client := &ec2Client{
		client: mockAPI,
	}

	// Define test data
	instance1ID := "i-1234567890abcdef0"
	instance2ID := "i-0987654321fedcba"
	
	// Setup mock expectations
	mockAPI.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
		return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instance1ID
	}), mock.Anything).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{
			Instances: []types.Instance{{
				InstanceId: &instance1ID,
				InstanceType: types.InstanceTypeT2Micro,
			}},
		}},
	}, nil)

	mockAPI.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
		return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instance2ID
	}), mock.Anything).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{
			Instances: []types.Instance{{
				InstanceId: &instance2ID,
				InstanceType: types.InstanceTypeT2Small,
			}},
		}},
	}, nil)

	// Test the function
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := client.GetInstanceConfigs(ctx, []string{instance1ID, instance2ID})

	// Collect and verify results
	var resultCount int
	for result := range results {
		assert.NoError(t, result.Err)
		assert.NotNil(t, result.Config)
		if result.InstanceID == instance1ID {
			assert.Equal(t, string(types.InstanceTypeT2Micro), result.Config.InstanceType)
		} else if result.InstanceID == instance2ID {
			assert.Equal(t, string(types.InstanceTypeT2Small), result.Config.InstanceType)
		} else {
			t.Errorf("Unexpected instance ID: %s", result.InstanceID)
		}
		resultCount++
	}

	assert.Equal(t, 2, resultCount)
}

// TestGetInstanceConfigs_Error tests error handling in GetInstanceConfigs
func TestGetInstanceConfigs_Error(t *testing.T) {
	// Setup mock
	mockAPI := &testutils.MockEC2API{}
	client := &ec2Client{
		client: mockAPI,
	}

	// Define test data
	instance1ID := "i-1234567890abcdef0"
	instance2ID := "i-0987654321fedcba"
	
	// Setup mock to return error for instance1 and success for instance2
	testErr := errors.New("test error")
	mockAPI.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
		return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instance1ID
	}), mock.Anything).Return(nil, testErr)

	mockAPI.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
		return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instance2ID
	}), mock.Anything).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{
			Instances: []types.Instance{{
				InstanceId: &instance2ID,
				InstanceType: types.InstanceTypeT2Micro,
			}},
		}},
	}, nil)

	// Test the function
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := client.GetInstanceConfigs(ctx, []string{instance1ID, instance2ID, ""})

	// Collect and verify results
	var (
		successCount int
		errorCount   int
	)

	for result := range results {
		if result.InstanceID == "" {
			errorCount++
			assert.Error(t, result.Err)
			assert.Contains(t, result.Err.Error(), "instance ID cannot be empty")
		} else if result.InstanceID == instance1ID {
			errorCount++
			assert.Error(t, result.Err)
			assert.Contains(t, result.Err.Error(), testErr.Error())
		} else if result.InstanceID == instance2ID {
			successCount++
			assert.NoError(t, result.Err)
			assert.NotNil(t, result.Config)
		}
	}

	assert.Equal(t, 1, successCount)
	assert.Equal(t, 2, errorCount)
}
