package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/driftdetector/internal/models"
)

// Helper functions for pointer values
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

// getWorkingDir gets the current working directory for test debugging
func getWorkingDir(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	return wd
}

func TestParseComplexHCL(t *testing.T) {
	// Test parsing a complex Terraform configuration file
	fullPath := filepath.Join("..", "..", "testdata", "terraform", "complex_instance.tf")
	
	// Check if the test file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Skipf("Skipping test because %s does not exist (cwd: %s)", fullPath, getWorkingDir(t))
	}
	
	t.Logf("Using test file: %s", fullPath)
	
	// Parse the file
	parser := NewParser()
	instance, err := parser.ParseHCL(fullPath)
	
	// Basic assertions
	assert.NoError(t, err, "ParseHCL should not return an error")
	assert.NotNil(t, instance, "Instance should not be nil")
	
	// Verify basic instance properties
	assert.Equal(t, "t3.medium", instance.InstanceType, "Instance type should be set from variable default")
	assert.Equal(t, "ami-0c55b159cbfafe1f0", instance.AMI, "AMI should be set from variable default")
	
	// Verify tags
	assert.NotNil(t, instance.Tags, "Tags should not be nil")
	assert.Equal(t, "test-web-server", instance.Tags["Name"], "Name tag should be set with environment variable")
	assert.Equal(t, "test", instance.Tags["Environment"], "Environment tag should be set from variable")
	assert.Equal(t, "terraform", instance.Tags["ManagedBy"], "ManagedBy tag should be set")
	
	// Verify root block device properties (stored as direct fields in InstanceConfig)
	assert.Equal(t, 30, instance.RootVolumeSize, "Root volume size should be set")
	assert.Equal(t, "gp3", instance.RootVolumeType, "Root volume type should be set")
	assert.True(t, *instance.RootVolumeEncrypted, "Root volume should be encrypted")
	
	// Verify metadata options
	assert.NotNil(t, instance.MetadataOptions, "Metadata options should not be nil")
	assert.Equal(t, "enabled", instance.MetadataOptions.HTTPEndpoint, "HTTP endpoint should be enabled")
	assert.Equal(t, "required", instance.MetadataOptions.HTTPTokens, "HTTP tokens should be required")
	assert.Equal(t, 1, *instance.MetadataOptions.HTTPPutResponseHopLimit, "HTTP put response hop limit should be 1")
	assert.Equal(t, "enabled", instance.MetadataOptions.InstanceMetadataTags, "Instance metadata tags should be enabled")
	
	// Verify credit specification
	assert.NotNil(t, instance.CreditSpecification, "Credit specification should not be nil")
	assert.Equal(t, "standard", instance.CreditSpecification.CPUCredits, "CPU credits should be standard")
	
	// Verify user data
	assert.NotEmpty(t, instance.UserData, "User data should not be empty")
}

func TestParseHCL(t *testing.T) {
	tests := []struct {
		name     string
		tfConfig string
		expected *models.InstanceConfig
		wantErr  bool
	}{
		{
			name: "basic instance with minimal configuration",
			tfConfig: `
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  
  tags = {
    Name = "example-instance"
  }
  
  vpc_security_group_ids = ["sg-12345678"]
}
`,
			expected: &models.InstanceConfig{
				InstanceType: "t2.micro",
				AMI:          "ami-0c55b159cbfafe1f0",
				Tags: map[string]string{
					"Name": "example-instance",
				},
				SecurityGroups: []models.SecurityGroup{},
			},
			wantErr: false,
		},
		{
			name: "instance with root block device",
			tfConfig: `
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  
  root_block_device {
    volume_size = 20
    volume_type = "gp3"
    iops       = 3000
    throughput = 125
    encrypted  = true
    kms_key_id = "alias/aws/ebs"
  }
}
`,
			expected: &models.InstanceConfig{
				InstanceType: "t3.micro",
				AMI:          "ami-0c55b159cbfafe1f0",
				Tags:         map[string]string{},
				SecurityGroups: []models.SecurityGroup{},
				RootVolumeSize:       20,
				RootVolumeType:       "gp3",
				RootVolumeIops:       3000,
				RootVolumeThroughput: 125,
				RootVolumeEncrypted:  boolPtr(true),
				RootVolumeKMSKeyID:   "alias/aws/ebs",
			},
			wantErr: false,
		},
		{
			name: "instance with variable references",
			tfConfig: `
variable "instance_type" {
  type    = string
  default = "t3.micro"
}

variable "ami_id" {
  type    = string
  default = "ami-0c55b159cbfafe1f0"
}

resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = var.instance_type
  
  tags = {
    Name = "example-instance"
  }
}
`,
			expected: &models.InstanceConfig{
				InstanceType: "t3.micro",
				AMI:          "ami-0c55b159cbfafe1f0",
				Tags: map[string]string{
					"Name": "example-instance",
				},
				SecurityGroups: []models.SecurityGroup{},
			},
			wantErr: false,
		},
		{
			name: "instance with undefined variable references",
			tfConfig: `
resource "aws_instance" "example" {
  ami           = var.undefined_ami
  instance_type = var.undefined_type
  
  tags = {
    Name = "example-instance"
  }
}
`,
			expected: &models.InstanceConfig{
				Tags: map[string]string{
					"Name": "example-instance",
				},
				SecurityGroups: []models.SecurityGroup{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test*.tf")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.WriteString(tt.tfConfig); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			p := &terraformParser{}
			got, err := p.ParseHCL(tmpfile.Name())

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHCL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
