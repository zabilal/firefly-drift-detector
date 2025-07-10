package models

// TerraformState represents the structure of a Terraform state file
type TerraformState struct {
	// Version is the Terraform state format version
	Version int `json:"version"`
	
	// TerraformVersion is the version of Terraform that created this state
	TerraformVersion string `json:"terraform_version"`
	
	// Serial is the state format version number
	Serial int64 `json:"serial"`
	
	// Lineage is a unique ID for the state
	Lineage string `json:"lineage"`
	
	// Outputs contains the outputs from the Terraform state
	Outputs map[string]TerraformOutput `json:"outputs"`
	
	// Resources contains the resources from the Terraform state
	Resources []TerraformResource `json:"resources"`
}

// TerraformOutput represents a Terraform output
type TerraformOutput struct {
	Sensitive bool        `json:"sensitive"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
}

// TerraformResource represents a Terraform resource
type TerraformResource struct {
	Module    string                  `json:"module"`
	Mode      string                  `json:"mode"`
	Type      string                  `json:"type"`
	Name      string                  `json:"name"`
	Provider  string                  `json:"provider"`
	Instances []TerraformResourceInstance `json:"instances"`
}

// TerraformResourceInstance represents an instance of a Terraform resource
type TerraformResourceInstance struct {
	SchemaVersion int                    `json:"schema_version"`
	Attributes   map[string]interface{} `json:"attributes"`
	// Add other fields as needed
}
