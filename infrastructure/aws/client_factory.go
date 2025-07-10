package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ClientFactory defines an interface for creating AWS service clients
type ClientFactory interface {
	// NewEC2Client creates a new EC2 client with the provided config
	NewEC2Client(cfg aws.Config) EC2API
}

// defaultClientFactory is the default implementation of ClientFactory
type defaultClientFactory struct{}

// NewClientFactory creates a new instance of the default client factory
func NewClientFactory() ClientFactory {
	return &defaultClientFactory{}
}

// NewEC2Client creates a new EC2 client with the provided config
func (f *defaultClientFactory) NewEC2Client(cfg aws.Config) EC2API {
	return ec2.NewFromConfig(cfg)
}
