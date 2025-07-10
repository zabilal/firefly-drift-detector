package models

import "encoding/json"

// Instance represents the domain model for an EC2 instance in our domain
// This is the aggregate root for instance-related operations
type Instance struct {
    // Basic instance information
    ID             string            `json:"instance_id"`
    Type           string            `json:"instance_type"`
    AMI            string            `json:"ami"`
    KeyName        string            `json:"key_name"`
    Tags           map[string]string `json:"tags"`
    
    // Networking
    VPCID                   string         `json:"vpc_id"`
    SubnetID                string         `json:"subnet_id"`
    SecurityGroups          []SecurityGroup `json:"security_groups"`
    PublicIPAddress         string         `json:"public_ip_address"`
    PrivateIPAddress        string         `json:"private_ip_address"`
    AssociatePublicIPAddress *bool         `json:"associate_public_ip,omitempty"`
    PrivateDNSName          string         `json:"private_dns_name"`
    PublicDNSName           string         `json:"public_dns_name"`
    
    // Storage
    RootVolumeSize          int            `json:"root_volume_size"`
    RootVolumeType          string         `json:"root_volume_type"`
    RootVolumeIops          int            `json:"root_volume_iops,omitempty"`
    RootVolumeEncrypted     *bool          `json:"root_volume_encrypted,omitempty"`
    
    // IAM and Monitoring
    IAMInstanceProfile      string         `json:"iam_instance_profile,omitempty"`
    Monitoring              *bool          `json:"monitoring,omitempty"`
    
    // Placement
    AvailabilityZone        string         `json:"availability_zone,omitempty"`
    Tenancy                string         `json:"tenancy,omitempty"`
    
    // Additional fields as needed...
}

// SecurityGroup represents a security group associated with an instance
type SecurityGroup struct {
    GroupID   string `json:"id"`
    GroupName string `json:"name,omitempty"`
}

// NewInstance creates a new Instance with required fields
func NewInstance(id, instanceType, ami string) *Instance {
    return &Instance{
        ID:      id,
        Type:    instanceType,
        AMI:     ami,
        Tags:    make(map[string]string),
    }
}

// AddTag adds a tag to the instance
func (i *Instance) AddTag(key, value string) {
    if i.Tags == nil {
        i.Tags = make(map[string]string)
    }
    i.Tags[key] = value
}

// IsValid checks if the instance has the minimum required fields
func (i *Instance) IsValid() bool {
    return i.ID != "" && i.Type != "" && i.AMI != ""
}

// Custom unmarshal to handle different tag formats
type instanceJSON struct {
    *Instance
    RawTags interface{} `json:"tags,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Instance
func (i *Instance) UnmarshalJSON(data []byte) error {
    var temp instanceJSON
    temp.Instance = i
    
    if err := json.Unmarshal(data, &temp); err != nil {
        return err
    }
    
    // Handle different tag formats
    if temp.RawTags != nil {
        switch v := temp.RawTags.(type) {
        case map[string]interface{}:
            i.Tags = make(map[string]string)
            for key, val := range v {
                if strVal, ok := val.(string); ok {
                    i.Tags[key] = strVal
                }
            }
        case []interface{}:
            // Handle array of tag objects if needed
            i.Tags = make(map[string]string)
            for _, item := range v {
                if tag, ok := item.(map[string]interface{}); ok {
                    if key, kOk := tag["Key"].(string); kOk {
                        if val, vOk := tag["Value"].(string); vOk {
                            i.Tags[key] = val
                        }
                    }
                }
            }
        }
    }
    
    return nil
}
