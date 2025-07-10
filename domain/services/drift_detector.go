package services

import (
	"fmt"
	"reflect"
	"strings"

	"driftdetector/domain/models"
)

// DriftDetector is a domain service that encapsulates the business logic
// for detecting configuration drift between actual and desired states
type DriftDetector struct {
	// ignoredFields are fields that should be excluded from drift detection
	ignoredFields map[string]bool
}

// NewDriftDetector creates a new instance of DriftDetector
func NewDriftDetector() *DriftDetector {
	return &DriftDetector{
		ignoredFields: map[string]bool{
			// Add fields that should be ignored during comparison
		},
	}
}

// CompareInstances compares two instances and returns a drift report
func (d *DriftDetector) CompareInstances(actual, desired *models.Instance) *models.DriftReport {
	report := models.NewDriftReport(actual.ID)

	// Use reflection to compare struct fields
	actualVal := reflect.ValueOf(actual).Elem()
	desiredVal := reflect.ValueOf(desired).Elem()

	d.compareStruct("", actualVal, desiredVal, report)

	return report
}

// compareStruct recursively compares struct fields
func (d *DriftDetector) compareStruct(prefix string, actual, expected reflect.Value, report *models.DriftReport) {
	// Implementation of struct comparison logic
	// This is a simplified version - you'll want to expand this
	// to handle all the different field types and edge cases

	if actual.Kind() != expected.Kind() {
		report.AddDrift(models.NewDrift(
			models.DriftTypeModified,
			prefix,
			actual.Interface(),
			expected.Interface(),
			"Type mismatch",
		))
		return
	}

	switch actual.Kind() {
	case reflect.Struct:
		for i := 0; i < actual.NumField(); i++ {
			fieldName := actual.Type().Field(i).Name
			fieldPath := prefix + "." + fieldName

			// Skip ignored fields
			if d.ignoredFields[fieldName] {
				continue
			}

			actualField := actual.Field(i)
			expectedField := expected.Field(i)

			d.compareStruct(fieldPath, actualField, expectedField, report)
		}

	case reflect.Map:
		d.compareMaps(prefix, actual, expected, report)

	case reflect.Slice, reflect.Array:
		d.compareSlices(prefix, actual, expected, report)

	default:
		if !reflect.DeepEqual(actual.Interface(), expected.Interface()) {
			report.AddDrift(models.NewDrift(
				models.DriftTypeModified,
				strings.TrimPrefix(prefix, "."),
				actual.Interface(),
				expected.Interface(),
				"Value mismatch",
			))
		}
	}
}

// compareMaps compares two map values
func (d *DriftDetector) compareMaps(prefix string, actual, expected reflect.Value, report *models.DriftReport) {
	// Implementation for comparing maps
	// This is a simplified version

	for _, key := range actual.MapKeys() {
		keyStr := key.String()
		actualValue := actual.MapIndex(key)
		expectedValue := expected.MapIndex(key)

		if !expectedValue.IsValid() {
			report.AddDrift(models.NewDrift(
				models.DriftTypeRemoved,
				prefix+"."+keyStr,
				nil,
				nil,
				"Field removed",
			))
			continue
		}

		if !reflect.DeepEqual(actualValue.Interface(), expectedValue.Interface()) {
			report.AddDrift(models.NewDrift(
				models.DriftTypeModified,
				prefix+"."+keyStr,
				actualValue.Interface(),
				expectedValue.Interface(),
				"Value modified",
			))
		}
	}

	// Check for added fields
	for _, key := range expected.MapKeys() {
		keyStr := key.String()
		if !actual.MapIndex(key).IsValid() {
			expectedValue := expected.MapIndex(key)
			report.AddDrift(models.NewDrift(
				models.DriftTypeAdded,
				prefix+"."+keyStr,
				nil,
				expectedValue.Interface(),
				"Field added",
			))
		}
	}
}

// compareSlices compares two slice/array values
func (d *DriftDetector) compareSlices(prefix string, actual, expected reflect.Value, report *models.DriftReport) {
	// Implementation for comparing slices/arrays
	// This is a simplified version

	if actual.Len() != expected.Len() {
		report.AddDrift(models.NewDrift(
			models.DriftTypeModified,
			prefix,
			actual.Interface(),
			expected.Interface(),
			"Length mismatch",
		))
		return
	}

	for i := 0; i < actual.Len(); i++ {
		d.compareStruct(fmt.Sprintf("%s[%d]", prefix, i), actual.Index(i), expected.Index(i), report)
	}
}
