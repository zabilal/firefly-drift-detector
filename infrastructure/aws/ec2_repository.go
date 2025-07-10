package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"driftdetector/domain/models"
	"driftdetector/domain/repositories"
)

// Ensure EC2Repository implements the InstanceRepository interface
var _ repositories.InstanceRepository = (*EC2Repository)(nil)

// EC2Repository implements the InstanceRepository interface for AWS EC2
type EC2Repository struct {
	client EC2API
}

// EC2API defines the interface for AWS EC2 operations we need
// This makes it easier to mock for testing
type EC2API interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
}

// NewEC2Repository creates a new EC2Repository with the provided EC2API client
func NewEC2Repository(client EC2API) *EC2Repository {
	if client == nil {
		panic("EC2API client cannot be nil")
	}
	return &EC2Repository{
		client: client,
	}
}

// GetByID retrieves an instance by its ID
func (r *EC2Repository) GetByID(ctx context.Context, id string) (*models.Instance, error) {
	if id == "" {
		return nil, fmt.Errorf("instance ID cannot be empty")
	}

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	}

	output, err := r.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance %s: %w", id, err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", id)
	}

	return r.convertToDomainInstance(ctx, output.Reservations[0].Instances[0])
}

// GetByIDs retrieves multiple instances by their IDs
func (r *EC2Repository) GetByIDs(ctx context.Context, ids []string) ([]*models.Instance, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one instance ID is required")
	}

	// AWS API has a limit of 1000 instance IDs per request
	const maxBatchSize = 1000
	var instances []*models.Instance

	for i := 0; i < len(ids); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		input := &ec2.DescribeInstancesInput{
			InstanceIds: batch,
		}

		output, err := r.client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				converted, err := r.convertToDomainInstance(ctx, instance)
				if err != nil {
					// Log the error but continue with other instances
					fmt.Printf("Warning: Failed to convert instance %s: %v\n", aws.ToString(instance.InstanceId), err)
					continue
				}
				instances = append(instances, converted)
			}
		}
	}

	return instances, nil
}

// FindAll retrieves all instances
func (r *EC2Repository) FindAll(ctx context.Context) ([]*models.Instance, error) {
	var instances []*models.Instance
	var nextToken *string

	for {
		input := &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		}

		output, err := r.client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, res := range output.Reservations {
			for _, instance := range res.Instances {
				converted, err := r.convertToDomainInstance(ctx, instance)
				if err != nil {
					// Log the error but continue with other instances
					fmt.Printf("Warning: Failed to convert instance %s: %v\n", aws.ToString(instance.InstanceId), err)
					continue
				}
				instances = append(instances, converted)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return instances, nil
}

// Save is not implemented as it's not needed for read-only operations
func (r *EC2Repository) Save(ctx context.Context, instance *models.Instance) error {
	return fmt.Errorf("not implemented")
}

// Delete is not implemented as it's not needed for read-only operations
func (r *EC2Repository) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

// getVolumeDetails fetches the details of an EBS volume by its ID
func (r *EC2Repository) getVolumeDetails(ctx context.Context, volumeID string) (*types.Volume, error) {
	if volumeID == "" {
		return nil, fmt.Errorf("volume ID cannot be empty")
	}

	input := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	}

	result, err := r.client.DescribeVolumes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe volume %s: %w", volumeID, err)
	}

	if len(result.Volumes) == 0 {
		return nil, fmt.Errorf("volume %s not found", volumeID)
	}

	return &result.Volumes[0], nil
}

// convertToDomainInstance converts an AWS EC2 instance to our domain model
func (r *EC2Repository) convertToDomainInstance(ctx context.Context, instance types.Instance) (*models.Instance, error) {
	// Create a new instance with basic information
	domainInstance := &models.Instance{
		ID:   aws.ToString(instance.InstanceId),
		Type: string(instance.InstanceType),
	}

	// Set the AMI if available
	if instance.ImageId != nil {
		domainInstance.AMI = *instance.ImageId
	}

	// Set the key name if available
	if instance.KeyName != nil {
		domainInstance.KeyName = *instance.KeyName
	}

	// Initialize tags map
	domainInstance.Tags = make(map[string]string)

	// Process tags
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			domainInstance.Tags[*tag.Key] = *tag.Value
		}
	}

	// Set networking information
	if instance.VpcId != nil {
		domainInstance.VPCID = *instance.VpcId
	}

	if instance.SubnetId != nil {
		domainInstance.SubnetID = *instance.SubnetId
	}

	if instance.PrivateIpAddress != nil {
		domainInstance.PrivateIPAddress = *instance.PrivateIpAddress
	}

	if instance.PublicIpAddress != nil {
		domainInstance.PublicIPAddress = *instance.PublicIpAddress
	}

	if instance.PrivateDnsName != nil {
		domainInstance.PrivateDNSName = *instance.PrivateDnsName
	}

	if instance.PublicDnsName != nil {
		domainInstance.PublicDNSName = *instance.PublicDnsName
	}

	// Set security groups
	if len(instance.SecurityGroups) > 0 {
		domainInstance.SecurityGroups = make([]models.SecurityGroup, 0, len(instance.SecurityGroups))
		for _, sg := range instance.SecurityGroups {
			if sg.GroupId != nil {
				domainInstance.SecurityGroups = append(domainInstance.SecurityGroups, models.SecurityGroup{
					GroupID:   *sg.GroupId,
					GroupName: aws.ToString(sg.GroupName),
				})
			}
		}
	}

	// Set root device information if available
	if instance.RootDeviceName != nil && len(instance.BlockDeviceMappings) > 0 {
		for _, bd := range instance.BlockDeviceMappings {
			if bd.DeviceName != nil && *bd.DeviceName == *instance.RootDeviceName && bd.Ebs != nil && bd.Ebs.VolumeId != nil {
				volume, err := r.getVolumeDetails(ctx, *bd.Ebs.VolumeId)
				if err != nil {
					// Log the error but continue with other instance data
					fmt.Printf("Warning: Failed to get volume details for %s: %v\n", *bd.Ebs.VolumeId, err)
					continue
				}

				// Set volume size if available
				if volume.Size != nil {
					domainInstance.RootVolumeSize = int(*volume.Size)
				}

				// Set volume type if available
				if volume.VolumeType != "" {
					domainInstance.RootVolumeType = string(volume.VolumeType)
				}

				// Set IOPS if available
				if volume.Iops != nil {
					domainInstance.RootVolumeIops = int(*volume.Iops)
				}

				// Set encryption status if available
				if volume.Encrypted != nil {
					encrypted := *volume.Encrypted
					domainInstance.RootVolumeEncrypted = &encrypted
				}

				break
			}
		}
	}

	return domainInstance, nil
}
