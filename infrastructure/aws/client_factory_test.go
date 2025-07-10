package aws_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	awsrepo "driftdetector/infrastructure/aws"
)

func TestNewClientFactory(t *testing.T) {
	// When
	factory := awsrepo.NewClientFactory()

	// Then
	assert.NotNil(t, factory, "Factory should not be nil")
}

func TestDefaultClientFactory_NewEC2Client(t *testing.T) {
	// Given
	factory := awsrepo.NewClientFactory()
	cfg := aws.Config{
		Region: "us-west-2",
	}

	// When
	ec2Client := factory.NewEC2Client(cfg)

	// Then
	assert.NotNil(t, ec2Client, "EC2 client should not be nil")
}
