package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourusername/driftdetector/internal/detector"
	"gopkg.in/yaml.v3"
)

// Formatter defines the interface for formatting drift reports
type Formatter interface {
	Format(report *detector.DriftReport) (string, error)
}

// FormatType represents the output format for the report
type FormatType string

const (
	// FormatJSON outputs the report in JSON format
	FormatJSON FormatType = "json"
	// FormatYAML outputs the report in YAML format
	FormatYAML FormatType = "yaml"
	// FormatText outputs the report in human-readable text format
	FormatText FormatType = "text"
)

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format FormatType) (Formatter, error) {
	switch format {
	case FormatJSON:
		return &jsonFormatter{}, nil
	case FormatYAML:
		return &yamlFormatter{}, nil
	case FormatText:
		return &textFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

type jsonFormatter struct{}

func (f *jsonFormatter) Format(report *detector.DriftReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("cannot format nil report")
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %v", err)
	}
	return string(data), nil
}

type yamlFormatter struct{}

func (f *yamlFormatter) Format(report *detector.DriftReport) (string, error) {
	data, err := yaml.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to YAML: %v", err)
	}
	return string(data), nil
}

type textFormatter struct{}

func (f *textFormatter) Format(report *detector.DriftReport) (string, error) {
	if report == nil {
		return "No report data available\n", nil
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Drift Detection Report\n"))
	sb.WriteString(fmt.Sprintf("Instance ID: %s\n", report.InstanceID))
	sb.WriteString(fmt.Sprintf("Drift Detected: %t\n", report.HasDrift))

	if !report.HasDrift {
		sb.WriteString("\nNo configuration drift detected.\n")
		return sb.String(), nil
	}

	sb.WriteString(fmt.Sprintf("\nFound %d drift(s):\n\n", len(report.Drifts)))

	for i, drift := range report.Drifts {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, drift.Type, drift.Path))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", drift.Description))

		switch drift.Type {
		case detector.DriftTypeAdded:
			sb.WriteString(fmt.Sprintf("   Actual: %v\n", formatValue(drift.Actual)))
		case detector.DriftTypeRemoved:
			sb.WriteString(fmt.Sprintf("   Expected: %v\n", formatValue(drift.Expected)))
		case detector.DriftTypeModified:
			sb.WriteString(fmt.Sprintf("   Actual: %v\n", formatValue(drift.Actual)))
			sb.WriteString(fmt.Sprintf("   Expected: %v\n", formatValue(drift.Expected)))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "<nil>"
	case string:
		if val == "" {
			return "<empty>"
		}
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
