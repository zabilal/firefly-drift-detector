package detector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/driftdetector/internal/detector"
	"github.com/yourusername/driftdetector/internal/models"
)

// Alias the package to make it clear we're using package-level constants
var (
	DriftTypeModified = detector.DriftTypeModified
	DriftTypeRemoved  = detector.DriftTypeRemoved
)

func TestDetectDrift_NoDrift(t *testing.T) {
	tests := []struct {
		name     string
		actual   *models.InstanceConfig
		expected *models.InstanceConfig
	}{
		{
			name:     "nil configurations",
			actual:   nil,
			expected: nil,
		},
		{
			name: "identical configurations",
			actual: &models.InstanceConfig{
				InstanceID:   "i-1234567890abcdef0",
				InstanceType: "t2.micro",
				AMI:          "ami-0c55b159cbfafe1f0",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				SecurityGroups: []models.SecurityGroup{
					{GroupID: "sg-12345678", GroupName: "test-sg"},
				},
			},
			expected: &models.InstanceConfig{
				InstanceID:   "i-1234567890abcdef0",
				InstanceType: "t2.micro",
				AMI:          "ami-0c55b159cbfafe1f0",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				SecurityGroups: []models.SecurityGroup{
					{GroupID: "sg-12345678", GroupName: "test-sg"},
				},
			},
		},
	}

	detector := detector.NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := detector.DetectDrift(tt.actual, tt.expected)
			if tt.actual != nil {
				assert.Equal(t, tt.actual.InstanceID, report.InstanceID)
			}
			assert.False(t, report.HasDrift)
			assert.Empty(t, report.Drifts)
		})
	}
}

func TestDetectDrift_Modified(t *testing.T) {
	actual := &models.InstanceConfig{
		InstanceID:   "i-1234567890abcdef0",
		InstanceType: "t2.micro",
		AMI:          "ami-0c55b159cbfafe1f0",
		Tags: map[string]string{
			"Name": "test-instance",
		},
	}

	expected := &models.InstanceConfig{
		InstanceID:   "i-1234567890abcdef0",
		InstanceType: "t2.medium", // Changed
		AMI:          "ami-0c55b159cbfafe1f0",
		Tags: map[string]string{
			"Name": "test-instance-modified", // Changed
		},
	}

	detector := detector.NewDetector()
	report := detector.DetectDrift(actual, expected)

	assert.True(t, report.HasDrift)
	assert.Len(t, report.Drifts, 2) // Both instance type and tag changed

	// Check instance type drift
	foundInstanceType := false
	foundTag := false

	for _, drift := range report.Drifts {
		switch drift.Path {
		case "InstanceType":
			foundInstanceType = true
			assert.Equal(t, DriftTypeModified, drift.Type)
			assert.Equal(t, "t2.micro", drift.Actual)
			assert.Equal(t, "t2.medium", drift.Expected)
		case "Tags[Name]":
			foundTag = true
			assert.Equal(t, DriftTypeModified, drift.Type)
			assert.Equal(t, "test-instance", drift.Actual)
			assert.Equal(t, "test-instance-modified", drift.Expected)
		}
	}

	assert.True(t, foundInstanceType, "Expected InstanceType drift not found")
	assert.True(t, foundTag, "Expected tag drift not found")
}

func TestDetectDrift_AddedRemoved(t *testing.T) {
	actual := &models.InstanceConfig{
		InstanceID:   "i-1234567890abcdef0",
		InstanceType: "t2.micro",
		// No tags in actual
	}

	expected := &models.InstanceConfig{
		InstanceID:   "i-1234567890abcdef0",
		InstanceType: "t2.micro",
		Tags: map[string]string{
			"Environment": "production",
		},
	}

	detector := detector.NewDetector()
	report := detector.DetectDrift(actual, expected)

	assert.True(t, report.HasDrift)
	assert.Len(t, report.Drifts, 1)

	drift := report.Drifts[0]
	assert.Equal(t, "Tags[Environment]", drift.Path)
	assert.Equal(t, DriftTypeRemoved, drift.Type)
	assert.Nil(t, drift.Actual)
	assert.Equal(t, "production", drift.Expected)
}

func TestDetectDrift_SecurityGroups(t *testing.T) {
	tests := []struct {
		name          string
		actualSGs     []models.SecurityGroup
		expectedSGs    []models.SecurityGroup
		expectedDrifts int
	}{
		{
			name: "no drift",
			actualSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test"},
			},
			expectedSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test"},
			},
			expectedDrifts: 0,
		},
		{
			name: "modified security group",
			actualSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test"},
			},
			expectedSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test-modified"},
			},
			expectedDrifts: 1, // Name changed
		},
		{
			name: "added security group",
			actualSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test"},
				{GroupID: "sg-2", GroupName: "test2"},
			},
			expectedSGs: []models.SecurityGroup{
				{GroupID: "sg-1", GroupName: "test"},
			},
			expectedDrifts: 1, // One group added
		},
	}

	detector := detector.NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := &models.InstanceConfig{
				InstanceID:    "i-1234567890abcdef0",
				InstanceType:  "t2.micro",
				SecurityGroups: tt.actualSGs,
			}

			expected := &models.InstanceConfig{
				InstanceID:    "i-1234567890abcdef0",
				InstanceType:  "t2.micro",
				SecurityGroups: tt.expectedSGs,
			}

			report := detector.DetectDrift(actual, expected)
			if tt.expectedDrifts > 0 {
				assert.True(t, report.HasDrift)
				assert.Len(t, report.Drifts, tt.expectedDrifts)
			} else {
				assert.False(t, report.HasDrift)
			}
		})
	}
}
