// Package aws provides functionality to interact with AWS EC2 instances and retrieve
// their configurations for drift detection purposes.
//
// This package includes clients and utilities to fetch EC2 instance details,
// handle AWS API interactions, and convert AWS-specific types to our internal models.
package aws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/yourusername/driftdetector/internal/models"
)

// EC2Client defines the interface for interacting with AWS EC2 instances.
// It abstracts the AWS SDK calls to make them more testable and maintainable.
// InstanceConfigResult represents the result of a GetInstanceConfig operation
// for a single instance, including either the configuration or an error.
type InstanceConfigResult struct {
	InstanceID string
	Config     *models.InstanceConfig
	Err        error
}

type EC2Client interface {
	// GetInstanceConfig retrieves the configuration of the specified EC2 instance.
	// It returns an InstanceConfig containing the instance details or an error if the operation fails.
	GetInstanceConfig(ctx context.Context, instanceID string) (*models.InstanceConfig, error)
	
	// GetInstanceConfigs retrieves configurations for multiple EC2 instances concurrently.
	// It returns a channel that will receive InstanceConfigResult for each instance.
	// The channel will be closed when all operations complete.
	GetInstanceConfigs(ctx context.Context, instanceIDs []string) <-chan InstanceConfigResult
}

// ec2Client implements the EC2Client interface using the AWS SDK v2
// It uses EC2DescribeInstancesAPI to make it more testable
type ec2Client struct {
	client EC2DescribeInstancesAPI
}

// NewEC2Client initializes and returns a new EC2 client with the specified AWS region.
// It uses the default AWS credential chain to authenticate with AWS.
//
// Parameters://   - ctx: Context for cancellation and timeouts
//   - region: AWS region to create the client for (e.g., "us-west-2")
//
// Returns:
//   - EC2Client: An implementation of the EC2Client interface
//   - error: If the client cannot be created
func NewEC2Client(ctx context.Context, region string) (EC2Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	return &ec2Client{
		client: ec2.NewFromConfig(cfg),
	}, nil
}

// GetInstanceConfig fetches the current configuration of an EC2 instance from AWS.
// It validates the instance ID and handles AWS API errors appropriately.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - instanceID: The ID of the EC2 instance to retrieve
//
// Returns:
//   - *models.InstanceConfig: The instance configuration if found
//   - error: If the instance cannot be found or an API error occurs
// GetInstanceConfigs retrieves configurations for multiple EC2 instances concurrently.
// It returns a channel that will receive InstanceConfigResult for each instance.
// The channel will be closed when all operations complete.
func (c *ec2Client) GetInstanceConfigs(ctx context.Context, instanceIDs []string) <-chan InstanceConfigResult {
	// Create a buffered channel to collect results
	results := make(chan InstanceConfigResult, len(instanceIDs))

	// Use a wait group to wait for all goroutines to complete
	var wg sync.WaitGroup
	
	// Process each instance ID in a separate goroutine
	for _, id := range instanceIDs {
		// Skip empty instance IDs
		if id == "" {
			results <- InstanceConfigResult{
				InstanceID: id,
				Err:        errors.New("instance ID cannot be empty"),
			}
			continue
		}

		wg.Add(1)
		go func(instanceID string) {
			defer wg.Done()

			// Create a new context with a timeout for each request
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Get the instance configuration
			config, err := c.GetInstanceConfig(ctx, instanceID)
			
			// Send the result to the channel
			results <- InstanceConfigResult{
				InstanceID: instanceID,
				Config:     config,
				Err:        err,
			}
		}(id)
	}

	// Close the results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func (c *ec2Client) GetInstanceConfig(ctx context.Context, instanceID string) (*models.InstanceConfig, error) {
	if instanceID == "" {
		return nil, errors.New("instance ID cannot be empty")
	}

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %v", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("no instance found with ID: %s", instanceID)
	}

	instance := result.Reservations[0].Instances[0]
	return convertToInstanceConfig(instance), nil
}

// convertToInstanceConfig converts an AWS EC2 instance to our internal InstanceConfig model.
// It extracts relevant fields and transforms them into a more structured format.
//
// Parameters:
//   - instance: The AWS SDK EC2 instance to convert
//
// Returns:
//   - *models.InstanceConfig: The converted instance configuration
func convertToInstanceConfig(instance types.Instance) *models.InstanceConfig {
	cfg := &models.InstanceConfig{
		InstanceID:       aws.ToString(instance.InstanceId),
		InstanceType:     string(instance.InstanceType),
		AMI:              aws.ToString(instance.ImageId),
		VPCID:            aws.ToString(instance.VpcId),
		SubnetID:         aws.ToString(instance.SubnetId),
		KeyName:          aws.ToString(instance.KeyName),
		PublicIPAddress:  aws.ToString(instance.PublicIpAddress),
		PrivateIPAddress: aws.ToString(instance.PrivateIpAddress),
		Tags:             make(map[string]string),
	}

	// Extract security groups
	for _, sg := range instance.SecurityGroups {
		cfg.SecurityGroups = append(cfg.SecurityGroups, models.SecurityGroup{
			GroupID:   aws.ToString(sg.GroupId),
			GroupName: aws.ToString(sg.GroupName),
		})
	}

	// Extract tags
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			cfg.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return cfg
}
