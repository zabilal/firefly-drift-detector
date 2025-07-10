package models

// DriftType represents the type of drift detected
type DriftType string

const (
    // DriftTypeAdded indicates a field exists in actual but not in expected
    DriftTypeAdded DriftType = "ADDED"
    // DriftTypeRemoved indicates a field exists in expected but not in actual
    DriftTypeRemoved DriftType = "REMOVED"
    // DriftTypeModified indicates a field exists in both but with different values
    DriftTypeModified DriftType = "MODIFIED"
)

// Drift represents a single drift finding in our domain
// This is a value object that's immutable once created
type Drift struct {
    Type        DriftType   `json:"type"`
    Path        string      `json:"path"`
    Actual      interface{} `json:"actual,omitempty"`
    Expected    interface{} `json:"expected,omitempty"`
    Description string      `json:"description"`
}

// NewDrift creates a new Drift value object
func NewDrift(driftType DriftType, path string, actual, expected interface{}, description string) Drift {
    return Drift{
        Type:        driftType,
        Path:        path,
        Actual:      actual,
        Expected:    expected,
        Description: description,
    }
}

// DriftReport represents the result of comparing two configurations
// This is an aggregate that contains all drift findings for a specific instance
type DriftReport struct {
    InstanceID string  `json:"instance_id"`
    HasDrift   bool    `json:"has_drift"`
    Drifts     []Drift `json:"drifts"`
}

// NewDriftReport creates a new DriftReport
func NewDriftReport(instanceID string) *DriftReport {
    return &DriftReport{
        InstanceID: instanceID,
        Drifts:     make([]Drift, 0),
    }
}

// AddDrift adds a new drift finding to the report
func (r *DriftReport) AddDrift(drift Drift) {
    r.Drifts = append(r.Drifts, drift)
    r.HasDrift = true
}

// GetDrifts returns all drifts in the report
func (r *DriftReport) GetDrifts() []Drift {
    return r.Drifts
}

// HasDrifts returns true if the report contains any drifts
func (r *DriftReport) HasDrifts() bool {
    return r.HasDrift
}
