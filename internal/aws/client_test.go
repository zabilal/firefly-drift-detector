package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yourusername/driftdetector/internal/aws/testutils"
	"github.com/yourusername/driftdetector/internal/logger"
)

func TestNewEC2ClientWrapper(t *testing.T) {
	tests := []struct {
		name          string
		region        string
		expectedError bool
	}{
		{
			name:          "valid region",
			region:        "us-west-2",
			expectedError: false,
		},
		{
			name:          "empty region",
			region:        "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewEC2ClientWrapper(context.Background(), tt.region)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestGetInstanceConfig_Success(t *testing.T) {
	// Setup mock
	mockSvc := new(testutils.MockEC2API)
	client := &EC2ClientWrapper{
		Client: mockSvc,
		logger: logger.NewLogger(logger.Config{Level: logger.LevelInfo}),
	}

	// Expected input/output
	expectedInstanceID := "i-1234567890abcdef0"
	expectedOutput := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{
			Instances: []types.Instance{{
				InstanceId:   aws.String(expectedInstanceID),
				InstanceType: "t2.micro",
				ImageId:      aws.String("ami-0c55b159cbfafe1f0"),
				VpcId:        aws.String("vpc-123456"),
				SubnetId:     aws.String("subnet-123456"),
				KeyName:      aws.String("my-key-pair"),
				State: &types.InstanceState{
					Name: types.InstanceStateNameRunning,
				},
				SecurityGroups: []types.GroupIdentifier{{
					GroupId:   aws.String("sg-123456"),
					GroupName: aws.String("my-security-group"),
				}},
				Tags: []types.Tag{{
					Key:   aws.String("Name"),
					Value: aws.String("test-instance"),
				}},
			}},
		}},
	}

	// Set expectations
	mockSvc.On("DescribeInstances", mock.Anything, &ec2.DescribeInstancesInput{
		InstanceIds: []string{expectedInstanceID},
	}).Return(expectedOutput, nil)

	// Execute
	config, err := client.GetInstanceConfig(context.Background(), expectedInstanceID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, expectedInstanceID, config.InstanceID)
	assert.Equal(t, "t2.micro", config.InstanceType)
	assert.Equal(t, "ami-0c55b159cbfafe1f0", config.AMI)
	assert.Len(t, config.SecurityGroups, 1)
	assert.Equal(t, "sg-123456", config.SecurityGroups[0].GroupID)
	assert.Equal(t, "test-instance", config.Tags["Name"])

	mockSvc.AssertExpectations(t)
}

func TestGetInstanceConfig_Error(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*testutils.MockEC2API)
		expectedErr string
	}{
		{
			name: "instance not found",
			setupMock: func(m *testutils.MockEC2API) {
				err := errors.New("InvalidInstanceID.NotFound: The instance ID 'i-1234567890abcdef0' does not exist")
				m.On("DescribeInstances", mock.Anything, mock.Anything, mock.Anything).
					Return((*ec2.DescribeInstancesOutput)(nil), err)
			},
			expectedErr: "failed to describe instance",
		},
		{
			name: "access denied",
			setupMock: func(m *testutils.MockEC2API) {
				err := errors.New("UnauthorizedOperation: You are not authorized to perform this operation")
				m.On("DescribeInstances", mock.Anything, mock.Anything, mock.Anything).
					Return((*ec2.DescribeInstancesOutput)(nil), err)
			},
			expectedErr: "access denied",
		},
		{
			name: "no instances found",
			setupMock: func(m *testutils.MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything, mock.Anything).
					Return(&ec2.DescribeInstancesOutput{Reservations: []types.Reservation{}}, nil)
			},
			expectedErr: "no instance found with ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockSvc := new(testutils.MockEC2API)
			tt.setupMock(mockSvc)

			client := &EC2ClientWrapper{
				Client: mockSvc,
				logger: logger.NewLogger(logger.Config{Level: logger.LevelError}),
			}

			// Execute
			config, err := client.GetInstanceConfig(context.Background(), "i-1234567890abcdef0")

			// Assert
			assert.Error(t, err)
			assert.Nil(t, config)
			assert.Contains(t, err.Error(), tt.expectedErr)

			// Verify expectations
			mockSvc.AssertExpectations(t)
		})
	}
}
