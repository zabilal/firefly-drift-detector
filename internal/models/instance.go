package models

import "encoding/json"

// InstanceConfig represents the configuration of an EC2 instance
type InstanceConfig struct {
    // Basic instance information
    InstanceID       string            `json:"instance_id"`
    InstanceType     string            `json:"instance_type"`
    AMI              string            `json:"ami"`
    KeyName          string            `json:"key_name"`
    Tags             map[string]string `json:"tags"`
    
    // RawTags is used for custom unmarshaling
    RawTags          interface{}       `json:"-"`
    
    // Networking
    VPCID                   string         `json:"vpc_id"`
    SubnetID                string         `json:"subnet_id"`
    SecurityGroups          []SecurityGroup `json:"security_groups"`
    PublicIPAddress         string         `json:"public_ip_address"`
    PrivateIPAddress        string         `json:"private_ip_address"`
    AssociatePublicIPAddress *bool         `json:"associate_public_ip,omitempty"`
    SourceDestCheck         *bool          `json:"source_dest_check,omitempty"`
    PrivateDNSName          string         `json:"private_dns_name"`
    PublicDNSName           string         `json:"public_dns_name"`
    
    // Storage
    RootVolumeSize          int            `json:"root_volume_size"`
    RootVolumeType          string         `json:"root_volume_type"`
    RootVolumeIops          int            `json:"root_volume_iops,omitempty"`
    RootVolumeThroughput    int            `json:"root_volume_throughput,omitempty"`
    RootVolumeEncrypted     *bool          `json:"root_volume_encrypted,omitempty"`
    RootVolumeKMSKeyID      string         `json:"root_volume_kms_key_id,omitempty"`
    EBSOptimized            *bool          `json:"ebs_optimized,omitempty"`
    
    // IAM and Monitoring
    IAMInstanceProfile      string         `json:"iam_instance_profile,omitempty"`
    Monitoring              *bool          `json:"monitoring,omitempty"`
    
    // Placement
    AvailabilityZone        string         `json:"availability_zone,omitempty"`
    Tenancy                string         `json:"tenancy,omitempty"`
    HostID                 string         `json:"host_id,omitempty"`
    PlacementGroup         string         `json:"placement_group,omitempty"`
    
    // CPU and Credits
    CPUCoreCount           *int           `json:"cpu_core_count,omitempty"`
    CPUThreadsPerCore      *int           `json:"cpu_threads_per_core,omitempty"`
    CreditSpecification    *CreditSpecification `json:"credit_specification,omitempty"`
    
    // Hibernation and Enclave
    Hibernation            *HibernationOptions `json:"hibernation,omitempty"`
    EnclaveOptions         *EnclaveOptions  `json:"enclave_options,omitempty"`
    
    // Metadata and User Data
    UserData               string         `json:"user_data,omitempty"`
    MetadataOptions        *MetadataOptions `json:"metadata_options,omitempty"`
    
    // Additional Configurations
    DisableAPITermination  *bool          `json:"disable_api_termination,omitempty"`
    InstanceInitiatedShutdownBehavior string `json:"instance_initiated_shutdown_behavior,omitempty"`
    
    // Launch Template
    LaunchTemplate         *LaunchTemplateSpecification `json:"launch_template,omitempty"`
    
    // Timeouts
    Timeouts               *Timeouts      `json:"timeouts,omitempty"`
    
    // Additional fields
    EBSBlockDevices       []*EBSBlockDevice       `json:"ebs_block_devices,omitempty"`
    EphemeralBlockDevices []*EphemeralBlockDevice `json:"ephemeral_block_devices,omitempty"`
    NetworkInterfaces     []*NetworkInterface      `json:"network_interfaces,omitempty"`
}

// tagPair represents a single key-value tag pair as used in AWS API responses
type tagPair struct {
    Key   string `json:"Key"`
    Value string `json:"Value"`
}

// UnmarshalJSON implements custom JSON unmarshaling for InstanceConfig
// to handle both map and array formats for tags
func (ic *InstanceConfig) UnmarshalJSON(data []byte) error {
    // Define a shadow type to avoid recursion
    type Alias InstanceConfig
    aux := &struct {
        *Alias
        RawTags interface{} `json:"tags"`
    }{
        Alias: (*Alias)(ic),
    }

    // Unmarshal the main structure
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }

    // Handle tags conversion
    if aux.RawTags != nil {
        switch v := aux.RawTags.(type) {
        case map[string]interface{}:
            // Tags are already in map format
            ic.Tags = make(map[string]string)
            for k, val := range v {
                if strVal, ok := val.(string); ok {
                    ic.Tags[k] = strVal
                }
            }
        case []interface{}:
            // Tags are in array format: [{"Key": "Name", "Value": "example"}]
            ic.Tags = make(map[string]string)
            for _, item := range v {
                if tagMap, ok := item.(map[string]interface{}); ok {
                    if key, keyOk := tagMap["Key"].(string); keyOk {
                        if val, valOk := tagMap["Value"].(string); valOk {
                            ic.Tags[key] = val
                        }
                    }
                }
            }
        }
    }

    return nil
}

// Supporting types
type SecurityGroup struct {
    GroupID   string `json:"id"`
    GroupName string `json:"name,omitempty"`
}

type CreditSpecification struct {
    CPUCredits string `json:"cpu_credits,omitempty"`
}

type HibernationOptions struct {
    Configured bool `json:"configured"`
}

type EnclaveOptions struct {
    Enabled bool `json:"enabled"`
}

type MetadataOptions struct {
    HTTPEndpoint            string `json:"http_endpoint,omitempty"`
    HTTPTokens              string `json:"http_tokens,omitempty"`
    HTTPPutResponseHopLimit *int   `json:"http_put_response_hop_limit,omitempty"`
    InstanceMetadataTags    string `json:"instance_metadata_tags,omitempty"`
}

type LaunchTemplateSpecification struct {
    ID      string `json:"id,omitempty"`
    Name    string `json:"name,omitempty"`
    Version string `json:"version,omitempty"`
}

type Timeouts struct {
    Create string `json:"create,omitempty"`
    Update string `json:"update,omitempty"`
    Delete string `json:"delete,omitempty"`
}

type EBSBlockDevice struct {
    DeviceName          string `json:"device_name"`
    SnapshotID          string `json:"snapshot_id,omitempty"`
    VolumeType          string `json:"volume_type,omitempty"`
    VolumeSize          *int   `json:"volume_size,omitempty"`
    Iops                *int   `json:"iops,omitempty"`
    DeleteOnTermination *bool  `json:"delete_on_termination,omitempty"`
    Encrypted           *bool  `json:"encrypted,omitempty"`
    KMSKeyID           string `json:"kms_key_id,omitempty"`
    Throughput          *int   `json:"throughput,omitempty"`
}

type EphemeralBlockDevice struct {
    DeviceName  string `json:"device_name"`
    NoDevice    *bool  `json:"no_device,omitempty"`
    VirtualName string `json:"virtual_name,omitempty"`
}

type NetworkInterface struct {
    DeviceIndex         int     `json:"device_index"`
    NetworkInterfaceID string  `json:"network_interface_id,omitempty"`
    DeleteOnTermination *bool   `json:"delete_on_termination,omitempty"`
    NetworkCardIndex    int     `json:"network_card_index,omitempty"`
}

// // InstanceConfig represents the configuration of an EC2 instance
// type InstanceConfig struct {
// 	InstanceID       string            `json:"instance_id"`
// 	InstanceType     string            `json:"instance_type"`
// 	AMI              string            `json:"ami"`
// 	VPCID            string            `json:"vpc_id"`
// 	SubnetID         string            `json:"subnet_id"`
// 	SecurityGroups   []SecurityGroup   `json:"security_groups"`
// 	Tags             map[string]string `json:"tags"`
// 	KeyName          string            `json:"key_name"`
// 	PublicIPAddress  string            `json:"public_ip_address"`
// 	PrivateIPAddress string            `json:"private_ip_address"`
// 	RootVolumeSize   int               `json:"root_volume_size"`   // Size in GB
// 	RootVolumeType   string            `json:"root_volume_type"`   // e.g., gp2, gp3, io1, etc.
// }

// // SecurityGroup represents a security group associated with an instance
// type SecurityGroup struct {
// 	GroupID   string `json:"group_id"`
// 	GroupName string `json:"group_name"`
// }
