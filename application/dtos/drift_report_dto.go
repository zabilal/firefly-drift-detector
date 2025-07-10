package dtos

import (
	"driftdetector/domain/models"
)

// DriftDTO represents a drift finding in the application layer
type DriftDTO struct {
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	Actual      interface{} `json:"actual,omitempty"`
	Expected    interface{} `json:"expected,omitempty"`
	Description string      `json:"description"`
}

// DriftReportDTO represents a drift report in the application layer
type DriftReportDTO struct {
	InstanceID string      `json:"instance_id"`
	HasDrift   bool        `json:"has_drift"`
	Drifts     []DriftDTO `json:"drifts"`
}

// NewDriftReportDTO creates a new DriftReportDTO from a domain model
func NewDriftReportDTO(report *models.DriftReport) *DriftReportDTO {
	if report == nil {
		return nil
	}

	drifts := make([]DriftDTO, len(report.Drifts))
	for i, d := range report.Drifts {
		drifts[i] = DriftDTO{
			Type:        string(d.Type),
			Path:        d.Path,
			Actual:      d.Actual,
			Expected:    d.Expected,
			Description: d.Description,
		}
	}

	return &DriftReportDTO{
		InstanceID: report.InstanceID,
		HasDrift:   report.HasDrift,
		Drifts:     drifts,
	}
}

// InstanceDTO represents an instance in the application layer
type InstanceDTO struct {
	ID             string            `json:"instance_id"`
	Type           string            `json:"instance_type"`
	AMI            string            `json:"ami"`
	KeyName        string            `json:"key_name,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	PublicIP       string            `json:"public_ip_address,omitempty"`
	PrivateIP      string            `json:"private_ip_address,omitempty"`
	VPCID          string            `json:"vpc_id,omitempty"`
	SubnetID       string            `json:"subnet_id,omitempty"`
	State          string            `json:"state,omitempty"`
	LaunchTime     string            `json:"launch_time,omitempty"`
	Monitoring     string            `json:"monitoring,omitempty"`
	RootDeviceName string            `json:"root_device_name,omitempty"`
	RootDeviceType string            `json:"root_device_type,omitempty"`
}

// NewInstanceDTO creates a new InstanceDTO from a domain model
func NewInstanceDTO(instance *models.Instance) *InstanceDTO {
	if instance == nil {
		return nil
	}

	return &InstanceDTO{
		ID:             instance.ID,
		Type:           instance.Type,
		AMI:            instance.AMI,
		KeyName:        instance.KeyName,
		Tags:           instance.Tags,
		PublicIP:       instance.PublicIPAddress,
		PrivateIP:      instance.PrivateIPAddress,
		VPCID:          instance.VPCID,
		SubnetID:       instance.SubnetID,
		State:          "", // You might want to add state to your domain model
		LaunchTime:     "", // You might want to add launch time to your domain model
		RootDeviceName: "", // You might want to add root device info to your domain model
	}
}
