package report_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/driftdetector/internal/detector"
	"github.com/yourusername/driftdetector/internal/report"
)

func TestFormatter_JSON(t *testing.T) {
	tests := []struct {
		name     string
		report   *detector.DriftReport
		expected string
		hasError bool
	}{
		{
			name:     "nil report",
			report:   nil,
			hasError: true,
		},
		{
			name: "empty report",
			report: &detector.DriftReport{
				InstanceID: "i-1234567890abcdef0",
				HasDrift:   false,
				Drifts:     []detector.Drift{},
			},
			expected: `{
  "instance_id": "i-1234567890abcdef0",
  "has_drift": false,
  "drifts": []
}`,
		},
		{
			name: "report with drifts",
			report: &detector.DriftReport{
				InstanceID: "i-1234567890abcdef0",
				HasDrift:   true,
				Drifts: []detector.Drift{
					{
						Type:        detector.DriftTypeModified,
						Path:        "InstanceType",
						Actual:      "t2.micro",
						Expected:    "t2.medium",
						Description: "Instance type has changed",
					},
				},
			},
			expected: `{
  "instance_id": "i-1234567890abcdef0",
  "has_drift": true,
  "drifts": [
    {
      "type": "MODIFIED",
      "path": "InstanceType",
      "actual": "t2.micro",
      "expected": "t2.medium",
      "description": "Instance type has changed"
    }
  ]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := report.NewFormatter(report.FormatJSON)
			assert.NoError(t, err)

			result, err := formatter.Format(tt.report)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expected, result)
			}
		})
	}
}

func TestFormatter_Text(t *testing.T) {
	tests := []struct {
		name     string
		report   *detector.DriftReport
		expected string
	}{
		{
			name:     "nil report",
			report:   nil,
			expected: "No report data available\n",
		},
		{
			name: "no drift",
			report: &detector.DriftReport{
				InstanceID: "i-1234567890abcdef0",
				HasDrift:   false,
				Drifts:     []detector.Drift{},
			},
			expected: `Drift Detection Report
Instance ID: i-1234567890abcdef0
Drift Detected: false

No configuration drift detected.
`,
		},
		{
			name: "with drifts",
			report: &detector.DriftReport{
				InstanceID: "i-1234567890abcdef0",
				HasDrift:   true,
				Drifts: []detector.Drift{
					{
						Type:        detector.DriftTypeModified,
						Path:        "InstanceType",
						Actual:      "t2.micro",
						Expected:    "t2.medium",
						Description: "Instance type has changed",
					},
					{
						Type:        detector.DriftTypeAdded,
						Path:        "Tags[Environment]",
						Actual:      "production",
						Description: "Tag Environment was added",
					},
				},
			},
			expected: `Drift Detection Report
Instance ID: i-1234567890abcdef0
Drift Detected: true

Found 2 drift(s):

1. [MODIFIED] InstanceType
   Description: Instance type has changed
   Actual: t2.micro
   Expected: t2.medium

2. [ADDED] Tags[Environment]
   Description: Tag Environment was added
   Actual: production

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := report.NewFormatter(report.FormatText)
			assert.NoError(t, err)

			result, err := formatter.Format(tt.report)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewFormatter_UnsupportedFormat(t *testing.T) {
	_, err := report.NewFormatter("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}
