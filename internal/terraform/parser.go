package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	hcljson "github.com/hashicorp/hcl/v2/json"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/yourusername/driftdetector/internal/models"
)

// findBlockByType finds a block by type in a slice of blocks
func findBlockByType(blocks hclsyntax.Blocks, typeName string) *hclsyntax.Block {
	for _, block := range blocks {
		if block.Type == typeName {
			return block
		}
	}
	return nil
}

// Parser is an interface for parsing Terraform configurations and state files
type Parser interface {
	ParseHCL(filePath string) (*models.InstanceConfig, error)
	ParseState(filePath string) ([]*models.InstanceConfig, error)
}

type terraformParser struct{}

// NewParser creates a new Terraform parser
func NewParser() Parser {
	return &terraformParser{}
}

// setInstanceAttribute sets the appropriate field on InstanceConfig based on the attribute name and value
func setInstanceAttribute(cfg *models.InstanceConfig, name string, val cty.Value) error {
	switch name {
	// Basic instance information
	case "instance_type":
		if val.Type() == cty.String {
			cfg.InstanceType = val.AsString()
		}
	case "ami":
		if val.Type() == cty.String {
			cfg.AMI = val.AsString()
		}
	case "key_name":
		if val.Type() == cty.String {
			cfg.KeyName = val.AsString()
		}

	// Networking
	case "subnet_id":
		if val.Type() == cty.String {
			cfg.SubnetID = val.AsString()
		}
	case "vpc_id":
		if val.Type() == cty.String {
			cfg.VPCID = val.AsString()
		}
	case "associate_public_ip_address":
		if val.Type() == cty.Bool {
			b := val.True()
			cfg.AssociatePublicIPAddress = &b
		}
	case "source_dest_check":
		if val.Type() == cty.Bool {
			b := val.True()
			cfg.SourceDestCheck = &b
		}

	// Storage
	case "ebs_optimized":
		if val.Type() == cty.Bool {
			b := val.True()
			cfg.EBSOptimized = &b
		}

	// IAM and Monitoring
	case "iam_instance_profile":
		if val.Type() == cty.String {
			cfg.IAMInstanceProfile = val.AsString()
		}
	case "monitoring":
		if val.Type() == cty.Bool {
			b := val.True()
			cfg.Monitoring = &b
		}

	// Placement
	case "availability_zone":
		if val.Type() == cty.String {
			cfg.AvailabilityZone = val.AsString()
		}
	case "tenancy":
		if val.Type() == cty.String {
			cfg.Tenancy = val.AsString()
		}
	case "placement_group":
		if val.Type() == cty.String {
			cfg.PlacementGroup = val.AsString()
		}

	// CPU and Credits
	case "cpu_core_count":
		if val.Type() == cty.Number {
			if f, _ := val.AsBigFloat().Int64(); f > 0 {
				i := int(f)
				cfg.CPUCoreCount = &i
			}
		}
	case "cpu_threads_per_core":
		if val.Type() == cty.Number {
			if f, _ := val.AsBigFloat().Int64(); f > 0 {
				i := int(f)
				cfg.CPUThreadsPerCore = &i
			}
		}

	// Hibernation and Enclave
	case "user_data":
		if val.Type() == cty.String {
			cfg.UserData = val.AsString()
		}
	case "user_data_base64":
		if val.Type() == cty.String {
			// For testing purposes, we'll just use the string as-is
			// In a real implementation, you might want to decode the base64 string
			cfg.UserData = val.AsString()
		}

	// Additional Configurations
	case "disable_api_termination":
		if val.Type() == cty.Bool {
			b := val.True()
			cfg.DisableAPITermination = &b
		}
	case "instance_initiated_shutdown_behavior":
		if val.Type() == cty.String {
			cfg.InstanceInitiatedShutdownBehavior = val.AsString()
		}

	// Tags
	case "tags":
		if val.Type().IsObjectType() || val.Type().IsMapType() {
			tags := make(map[string]string)
			val.ForEachElement(func(key, val cty.Value) (stop bool) {
				if key.Type() == cty.String && val.Type() == cty.String {
					tags[key.AsString()] = val.AsString()
				}
				return false
			})
			if len(tags) > 0 {
				cfg.Tags = tags
			}
		}

	// Security Groups
	case "vpc_security_group_ids", "security_groups":
		if val.Type().IsListType() || val.Type().IsSetType() {
			it := val.ElementIterator()
			for it.Next() {
				_, v := it.Element()
				if v.Type() == cty.String {
					cfg.SecurityGroups = append(cfg.SecurityGroups, models.SecurityGroup{
						GroupID: v.AsString(),
					})
				}
			}
		}
	}

	return nil
}

// processRootBlockDevice processes the root_block_device nested block
func processRootBlockDevice(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "volume_size":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Float64(); f > 0 {
					cfg.RootVolumeSize = int(f)
				}
			}
		case "volume_type":
			if val.Type() == cty.String {
				cfg.RootVolumeType = val.AsString()
			}
		case "iops":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f > 0 {
					cfg.RootVolumeIops = int(f)
				}
			}
		case "throughput":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f > 0 {
					cfg.RootVolumeThroughput = int(f)
				}
			}
		case "encrypted":
			if val.Type() == cty.Bool {
				encrypted := val.True()
				cfg.RootVolumeEncrypted = &encrypted
			}
		case "kms_key_id":
			if val.Type() == cty.String {
				cfg.RootVolumeKMSKeyID = val.AsString()
			}
		}
	}
}

// processEBSBlockDevice processes an ebs_block_device nested block
func processEBSBlockDevice(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	ebsDevice := &models.EBSBlockDevice{}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "device_name":
			if val.Type() == cty.String {
				ebsDevice.DeviceName = val.AsString()
			}
		case "volume_size":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Float64(); f > 0 {
					size := int(f)
					ebsDevice.VolumeSize = &size
				}
			}
		case "volume_type":
			if val.Type() == cty.String {
				ebsDevice.VolumeType = val.AsString()
			}
		case "iops":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f > 0 {
					iops := int(f)
					ebsDevice.Iops = &iops
				}
			}
		case "delete_on_termination":
			if val.Type() == cty.Bool {
				deleteOnTerm := val.True()
				ebsDevice.DeleteOnTermination = &deleteOnTerm
			}
		case "encrypted":
			if val.Type() == cty.Bool {
				encrypted := val.True()
				ebsDevice.Encrypted = &encrypted
			}
		case "kms_key_id":
			if val.Type() == cty.String {
				ebsDevice.KMSKeyID = val.AsString()
			}
		case "throughput":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f > 0 {
					throughput := int(f)
					ebsDevice.Throughput = &throughput
				}
			}
		}
	}

	if ebsDevice.DeviceName != "" {
		cfg.EBSBlockDevices = append(cfg.EBSBlockDevices, ebsDevice)
	}
}

// processEphemeralBlockDevice processes an ephemeral_block_device nested block
func processEphemeralBlockDevice(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	ephemeralDevice := &models.EphemeralBlockDevice{}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "device_name":
			if val.Type() == cty.String {
				ephemeralDevice.DeviceName = val.AsString()
			}
		case "no_device":
			if val.Type() == cty.Bool {
				noDevice := val.True()
				ephemeralDevice.NoDevice = &noDevice
			}
		case "virtual_name":
			if val.Type() == cty.String {
				ephemeralDevice.VirtualName = val.AsString()
			}
		}
	}

	if ephemeralDevice.DeviceName != "" {
		cfg.EphemeralBlockDevices = append(cfg.EphemeralBlockDevices, ephemeralDevice)
	}
}

// processNetworkInterface processes a network_interface nested block
func processNetworkInterface(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	iface := &models.NetworkInterface{}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "device_index":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f >= 0 {
					iface.DeviceIndex = int(f)
				}
			}
		case "network_interface_id":
			if val.Type() == cty.String {
				iface.NetworkInterfaceID = val.AsString()
			}
		case "delete_on_termination":
			if val.Type() == cty.Bool {
				deleteOnTerm := val.True()
				iface.DeleteOnTermination = &deleteOnTerm
			}
		case "network_card_index":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f >= 0 {
					iface.NetworkCardIndex = int(f)
				}
			}
		}
	}

	// Only add if we have a valid device index
	if iface.DeviceIndex >= 0 {
		cfg.NetworkInterfaces = append(cfg.NetworkInterfaces, iface)
	}
}

// processCreditSpecification processes a credit_specification nested block
func processCreditSpecification(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	if cfg.CreditSpecification == nil {
		cfg.CreditSpecification = &models.CreditSpecification{}
	}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "cpu_credits":
			if val.Type() == cty.String {
				cfg.CreditSpecification.CPUCredits = val.AsString()
			}
		}
	}
}

// processMetadataOptions processes a metadata_options nested block
func processMetadataOptions(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	if cfg.MetadataOptions == nil {
		cfg.MetadataOptions = &models.MetadataOptions{}
	}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		switch name {
		case "http_endpoint":
			if val.Type() == cty.String {
				cfg.MetadataOptions.HTTPEndpoint = val.AsString()
			}
		case "http_tokens":
			if val.Type() == cty.String {
				cfg.MetadataOptions.HTTPTokens = val.AsString()
			}
		case "http_put_response_hop_limit":
			if val.Type() == cty.Number {
				if f, _ := val.AsBigFloat().Int64(); f >= 0 {
					hopLimit := int(f)
					cfg.MetadataOptions.HTTPPutResponseHopLimit = &hopLimit
				}
			}
		case "instance_metadata_tags":
			if val.Type() == cty.String {
				cfg.MetadataOptions.InstanceMetadataTags = val.AsString()
			}
		}
	}
}

// processEnclaveOptions processes an enclave_options nested block
func processEnclaveOptions(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	enabled := false

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		if name == "enabled" && val.Type() == cty.Bool {
			enabled = val.True()
		}
	}

	if cfg.EnclaveOptions == nil {
		cfg.EnclaveOptions = &models.EnclaveOptions{}
	}
	cfg.EnclaveOptions.Enabled = enabled
}

// processHibernationOptions processes a hibernation_options nested block
func processHibernationOptions(block *hclsyntax.Block, cfg *models.InstanceConfig, ctx *hcl.EvalContext) {
	configured := false

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			continue
		}

		if name == "configured" && val.Type() == cty.Bool {
			configured = val.True()
		}
	}

	if cfg.Hibernation == nil {
		cfg.Hibernation = &models.HibernationOptions{}
	}
	cfg.Hibernation.Configured = configured
}

// evalDefaultExpr evaluates the default value expression in a variable block
func evalDefaultExpr(block *hclsyntax.Block, ctx *hcl.EvalContext) (cty.Value, error) {
	if block == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}

	// Find the default attribute
	for _, attr := range block.Body.Attributes {
		if attr.Name == "default" {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("error evaluating default value: %v", diags)
			}
			return val, nil
		}
	}

	// No default value found
	return cty.NullVal(cty.DynamicPseudoType), nil
}

// loadVariables loads variable definitions from .tf files in the same directory
func loadVariables(dir string) (map[string]cty.Value, error) {
	variables := make(map[string]cty.Value)
	ctx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: map[string]function.Function{},
	}

	// Look for .tf files in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	// First pass: collect all variable declarations
	var varBlocks []*hclsyntax.Block

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".tf") {
			filePath := filepath.Join(dir, file.Name())
			src, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
			}

			file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				return nil, fmt.Errorf("failed to parse %s: %v", filePath, diags)
			}

			// Convert to hclsyntax.Body for easier traversal
			hclBody, ok := file.Body.(*hclsyntax.Body)
			if !ok {
				continue
			}

			// Collect variable blocks
			for _, block := range hclBody.Blocks {
				if block.Type == "variable" && len(block.Labels) > 0 {
					varBlocks = append(varBlocks, block)
				}
			}
		}
	}

	// Second pass: evaluate variables with access to other variables
	for _, block := range varBlocks {
		varName := block.Labels[0]
		
		// Get the default value
		val, err := evalDefaultExpr(block, ctx)
		if err != nil {
			fmt.Printf("Warning: Failed to evaluate default value for variable %s: %v\n", varName, err)
			val = cty.NullVal(cty.DynamicPseudoType)
		}

		// Store the variable in the context for subsequent evaluations
		variables[varName] = val
		ctx.Variables[varName] = val
	}

	return variables, nil
}

// resolveVariableReferences resolves variable references in an expression
func resolveVariableReferences(expr hcl.Expression, ctx *hcl.EvalContext) (hcl.Expression, error) {
	if expr == nil {
		return nil, nil
	}

	syntax, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return expr, nil
	}

	// Check if this is a variable reference (starts with var.)
	if len(syntax.Traversal) < 2 || syntax.Traversal.RootName() != "var" {
		return expr, nil
	}

	// Get the variable name (the part after var.)
	varName := syntax.Traversal[1].(hcl.TraverseAttr).Name

	// Look up the variable in the context
	val, exists := ctx.Variables["var"].AsValueMap()[varName]
	if !exists || val.IsNull() {
		return nil, fmt.Errorf("variable %s not found", varName)
	}

	// Return a literal expression with the variable's value
	return &hclsyntax.LiteralValueExpr{
		Val: val,
	}, nil
}

// evaluateExpression safely evaluates an HCL expression with a given context
func evaluateExpression(expr hcl.Expression, ctx *hcl.EvalContext) (cty.Value, error) {
	if expr == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}

	// First try to resolve any variable references
	resolved, err := resolveVariableReferences(expr, ctx)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return cty.NullVal(cty.DynamicPseudoType), nil
	}
	if resolved == nil {
		resolved = expr
	}

	// Now evaluate the expression
	val, diags := resolved.Value(ctx)
	if diags.HasErrors() {
		// Check if this is an undefined variable error
		isUndefinedVar := false
		for _, diag := range diags {
			if strings.Contains(diag.Error(), "Unknown variable") {
				isUndefinedVar = true
				break
			}
		}

		if isUndefinedVar {
			return cty.NullVal(cty.DynamicPseudoType), nil
		}
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("error evaluating expression: %v", diags)
	}

	return val, nil
}

// ParseHCL parses a Terraform HCL configuration file
func (p *terraformParser) ParseHCL(filePath string) (*models.InstanceConfig, error) {
	// Read the file content
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	ext := filepath.Ext(filePath)
	switch ext {
	case ".tf":
		// Parse the HCL file using hclsyntax
		file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
		if file == nil {
			return nil, fmt.Errorf("failed to parse HCL: no file was parsed")
		}
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL: %v", diags)
		}

		// Load variable definitions from the directory
		varDir := filepath.Dir(filePath)
		variables, err := loadVariables(varDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load variables: %v", err)
		}

		// Create an evaluation context with variables and functions
		ctx := &hcl.EvalContext{
			Variables: map[string]cty.Value{
				"var": cty.ObjectVal(variables),
			},
			Functions: map[string]function.Function{
				"base64encode": function.New(&function.Spec{
					Params: []function.Parameter{
						{
							Name: "str",
							Type: cty.String,
						},
					},
					Type: function.StaticReturnType(cty.String),
					Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
						if len(args) != 1 {
							return cty.NilVal, fmt.Errorf("base64encode takes exactly one argument")
						}
						str := args[0].AsString()
						return cty.StringVal(str), nil // Return the string as-is for testing
					},
				}),
			},
		}

		// Convert to hclsyntax.Body for easier traversal
		hclBody, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			return nil, fmt.Errorf("unexpected body type: %T", file.Body)
		}

		// Find AWS instance resources
		for _, block := range hclBody.Blocks {
			if block.Type == "resource" && len(block.Labels) >= 2 && block.Labels[0] == "aws_instance" {
				instanceName := block.Labels[1]
				fmt.Printf("Found AWS instance resource: %s\n", instanceName)

				// Create a new instance config with default values
				cfg := &models.InstanceConfig{
					InstanceID:    "", // Will be set by AWS when created
					Tags:          make(map[string]string),
					SecurityGroups: make([]models.SecurityGroup, 0),
				}

				// Process all attributes in the resource
				for name, attr := range block.Body.Attributes {
					val, err := evaluateExpression(attr.Expr, ctx)
					if err != nil {
						fmt.Printf("Warning: Failed to evaluate attribute %s: %v\n", name, err)
						continue
					}

					// Only process non-null values
					if !val.IsNull() {
						if err := setInstanceAttribute(cfg, name, val); err != nil {
							fmt.Printf("Warning: Failed to set attribute %s: %v\n", name, err)
						}
					}
				}

				// Process nested blocks
				for _, nestedBlock := range block.Body.Blocks {
					switch nestedBlock.Type {
					case "root_block_device":
						processRootBlockDevice(nestedBlock, cfg, ctx)

					case "ebs_block_device":
						processEBSBlockDevice(nestedBlock, cfg, ctx)

					case "ephemeral_block_device":
						processEphemeralBlockDevice(nestedBlock, cfg, ctx)

					case "network_interface":
						processNetworkInterface(nestedBlock, cfg, ctx)

					case "credit_specification":
						processCreditSpecification(nestedBlock, cfg, ctx)

					case "metadata_options":
						processMetadataOptions(nestedBlock, cfg, ctx)

					case "enclave_options":
						processEnclaveOptions(nestedBlock, cfg, ctx)

					case "hibernation_options":
						processHibernationOptions(nestedBlock, cfg, ctx)
					}
				}

				// Return the first instance found
				return cfg, nil
			}
		}

		return nil, fmt.Errorf("no AWS instance found in file %s", filePath)

	case ".tf.json":
		_, diags := hcljson.Parse(src, filePath)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse JSON: %v", diags)
		}
		// TODO: Implement JSON parsing for .tf.json files
		return nil, fmt.Errorf("parsing of .tf.json files is not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

// parseHCLInstance extracts instance configuration from HCL content
func parseHCLInstance(instance map[string]interface{}) *models.InstanceConfig {
	cfg := &models.InstanceConfig{
		Tags: make(map[string]string),
	}

	// Extract basic instance information
	if v, ok := instance["instance_type"]; ok {
		if instanceType, ok := v.(string); ok {
			cfg.InstanceType = instanceType
		}
	}

	if v, ok := instance["ami"]; ok {
		if ami, ok := v.(string); ok {
			cfg.AMI = ami
		}
	}

	if v, ok := instance["key_name"]; ok {
		if keyName, ok := v.(string); ok {
			cfg.KeyName = keyName
		}
	}

	// Extract networking information
	if v, ok := instance["subnet_id"]; ok {
		if subnetID, ok := v.(string); ok {
			cfg.SubnetID = subnetID
		}
	}

	if v, ok := instance["vpc_id"]; ok {
		if vpcID, ok := v.(string); ok {
			cfg.VPCID = vpcID
		}
	}

	// Extract security groups
	if v, ok := instance["vpc_security_group_ids"]; ok {
		if sgIDs, ok := v.([]interface{}); ok {
			for _, sgID := range sgIDs {
				if sgStr, ok := sgID.(string); ok {
					cfg.SecurityGroups = append(cfg.SecurityGroups, models.SecurityGroup{
						GroupID: sgStr,
					})
				}
			}
		}
	}

	// Extract tags
	if v, ok := instance["tags"]; ok {
		if tags, ok := v.(map[string]interface{}); ok {
			for k, v := range tags {
				if strVal, ok := v.(string); ok {
					cfg.Tags[k] = strVal
				}
			}
		}
	}

	// Extract root block device settings
	if v, ok := instance["root_block_device"]; ok {
		if rootBlockDevices, ok := v.([]interface{}); ok && len(rootBlockDevices) > 0 {
			if rootBlockDevice, ok := rootBlockDevices[0].(map[string]interface{}); ok {
				if volumeSize, ok := rootBlockDevice["volume_size"].(float64); ok {
					cfg.RootVolumeSize = int(volumeSize)
				}
				if volumeType, ok := rootBlockDevice["volume_type"].(string); ok {
					cfg.RootVolumeType = volumeType
				}
				if iops, ok := rootBlockDevice["iops"].(float64); ok {
					iopsInt := int(iops)
					cfg.RootVolumeIops = iopsInt
				}
				if throughput, ok := rootBlockDevice["throughput"].(float64); ok {
					throughputInt := int(throughput)
					cfg.RootVolumeThroughput = throughputInt
				}
				if encrypted, ok := rootBlockDevice["encrypted"].(bool); ok {
					cfg.RootVolumeEncrypted = &encrypted
				}
				if kmsKeyID, ok := rootBlockDevice["kms_key_id"].(string); ok {
					cfg.RootVolumeKMSKeyID = kmsKeyID
				}
			}
		}
	}

	// Extract EBS optimization
	if v, ok := instance["ebs_optimized"]; ok {
		ebsOptimized, ok := v.(bool)
		if ok {
			cfg.EBSOptimized = &ebsOptimized
		}
	}

	// Extract monitoring
	if v, ok := instance["monitoring"]; ok {
		monitoring, ok := v.(bool)
		if ok {
			cfg.Monitoring = &monitoring
		}
	}

	// Extract IAM instance profile
	if v, ok := instance["iam_instance_profile"]; ok {
		if iamProfile, ok := v.(string); ok {
			cfg.IAMInstanceProfile = iamProfile
		}
	}

	// Extract placement information
	if v, ok := instance["availability_zone"]; ok {
		if az, ok := v.(string); ok {
			cfg.AvailabilityZone = az
		}
	}

	if v, ok := instance["tenancy"]; ok {
		if tenancy, ok := v.(string); ok {
			cfg.Tenancy = tenancy
		}
	}

	if v, ok := instance["placement_group"]; ok {
		if placementGroup, ok := v.(string); ok {
			cfg.PlacementGroup = placementGroup
		}
	}

	return cfg
}

// ParseState parses a Terraform state file and returns a list of instance configurations
func (p *terraformParser) ParseState(filePath string) ([]*models.InstanceConfig, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	stateContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	// Debug: Print the raw content
	fmt.Println("=== Raw State File ===")
	fmt.Println(string(stateContent))
	fmt.Println("=====================")

	var state tfjson.State
	if err := json.Unmarshal(stateContent, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %v", err)
	}

	// Debug: Print the parsed state structure
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("=== Parsed State ===\n%s\n==================\n", string(stateJSON))

	if state.Values == nil {
		return nil, fmt.Errorf("state.Values is nil")
	}

	if state.Values.RootModule.Resources == nil {
		return nil, fmt.Errorf("state.Values.RootModule.Resources is nil")
	}

	var allConfigs []*models.InstanceConfig

	// Process all AWS instance resources
	for i, resource := range state.Values.RootModule.Resources {
		if resource == nil {
			fmt.Printf("Warning: Resource at index %d is nil\n", i)
			continue
		}

		fmt.Printf("Resource %d: Type=%s, Name=%s\n", i, resource.Type, resource.Name)
		if resource.Type == "aws_instance" {
			fmt.Printf("Found AWS instance resource: %s\n", resource.Name)
			configs, err := parseInstanceResource(resource)
			if err != nil {
				fmt.Printf("Error parsing instance resource %s: %v\n", resource.Name, err)
				continue
			}
			allConfigs = append(allConfigs, configs...)
		}

		// Check nested modules
		for j, module := range state.Values.RootModule.ChildModules {
			if module == nil {
				fmt.Printf("  Warning: Module at index %d is nil\n", j)
				continue
			}

			fmt.Printf("  Module %d: Address=%s\n", j, module.Address)
			// Process all resources in the module
			for k, r := range module.Resources {
				if r == nil {
					fmt.Printf("    Warning: Resource at index %d is nil\n", k)
					continue
				}

				fmt.Printf("    Resource %d: Type=%s, Name=%s\n", k, r.Type, r.Name)
				if r.Type == "aws_instance" {
					fmt.Printf("    Found AWS instance resource in module: %s\n", r.Name)
					configs, err := parseInstanceResource(r)
					if err != nil {
						fmt.Printf("Error parsing instance resource %s in module: %v\n", r.Name, err)
						continue
					}
					allConfigs = append(allConfigs, configs...)
				}
			}
		}
	}

	if len(allConfigs) == 0 {
		return nil, fmt.Errorf("no AWS instances found in state file")
	}

	return allConfigs, nil
}

// parseInstanceResource extracts instance configuration from a Terraform state resource
func parseInstanceResource(resource *tfjson.StateResource) ([]*models.InstanceConfig, error) {
	if resource == nil {
		return nil, fmt.Errorf("resource is nil")
	}

	// Debug: Print the resource to understand its structure
	fmt.Printf("=== Resource ===\nType: %s\nName: %s\nProvider: %s\nMode: %s\nAttributeValues: %+v\n==========\n", 
		resource.Type, resource.Name, resource.ProviderName, resource.Mode, resource.AttributeValues)

	// For now, we'll treat each resource as a single instance
	// In the future, we can enhance this to handle count and for_each
	cfg := &models.InstanceConfig{
		Tags: make(map[string]string),
	}

	// Extract instance ID from AttributeValues
	var instanceID string
	foundID := false

	// 1. Try to get instance ID from AttributeValues["id"]
	if id, ok := resource.AttributeValues["id"].(string); ok && id != "" {
		instanceID = id
		foundID = true
		fmt.Printf("Found instance ID from AttributeValues[id]: %s\n", instanceID)
	}

	// 2. Try to get instance ID from AttributeValues["instance_id"]
	if !foundID {
		if id, ok := resource.AttributeValues["instance_id"].(string); ok && id != "" {
			instanceID = id
			foundID = true
			fmt.Printf("Found instance ID from AttributeValues[instance_id]: %s\n", instanceID)
		}
	}

	if !foundID {
		return nil, fmt.Errorf("could not determine instance ID from resource")
	}

	cfg.InstanceID = instanceID

	// Extract instance type
	if instanceType, ok := resource.AttributeValues["instance_type"].(string); ok && instanceType != "" {
		cfg.InstanceType = instanceType
	}

	// Extract AMI
	if ami, ok := resource.AttributeValues["ami"].(string); ok && ami != "" {
		cfg.AMI = ami
	}

	// Extract tags
	if tags, ok := resource.AttributeValues["tags"].(map[string]interface{}); ok {
		for k, v := range tags {
			if strVal, ok := v.(string); ok {
				cfg.Tags[k] = strVal
			}
		}
	}

	// Extract security groups
	if sgIDs, ok := resource.AttributeValues["vpc_security_group_ids"].([]interface{}); ok {
		for _, sg := range sgIDs {
			if sgStr, ok := sg.(string); ok {
				cfg.SecurityGroups = append(cfg.SecurityGroups, models.SecurityGroup{
					GroupID:   sgStr,
					GroupName: "", // We don't have this info in the state
				})
			}
		}
	}

	// Extract subnet ID
	if subnetID, ok := resource.AttributeValues["subnet_id"].(string); ok && subnetID != "" {
		cfg.SubnetID = subnetID
	}

	// Extract key name
	if keyName, ok := resource.AttributeValues["key_name"].(string); ok && keyName != "" {
		cfg.KeyName = keyName
	}

	// Extract root block device settings
	if rootBlockDevices, ok := resource.AttributeValues["root_block_device"].([]interface{}); ok && len(rootBlockDevices) > 0 {
		if rootBlockDevice, ok := rootBlockDevices[0].(map[string]interface{}); ok {
			if volumeSize, ok := rootBlockDevice["volume_size"].(float64); ok {
				cfg.RootVolumeSize = int(volumeSize)
			}
			if volumeType, ok := rootBlockDevice["volume_type"].(string); ok {
				cfg.RootVolumeType = volumeType
			}
		}
	}

	return []*models.InstanceConfig{cfg}, nil
}
