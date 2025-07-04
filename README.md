# Firefly AWS EC2 & Terraform Drift Detection Tool

A Go-based command-line tool to detect configuration drift between AWS EC2 instances and their corresponding Terraform configurations.

## Features

- **CLI Interface**: Easy-to-use command-line interface
- **Drift Detection**: Compare AWS EC2 instances with Terraform state/config
- **Multiple Formats**: Support for both Terraform state files and directories
- **Concurrent Processing**: Check multiple instances in parallel
- **Detailed Reports**: Clear, structured output of configuration differences
- **Versioning**: Built-in version tracking

## 🚀 Installation

### Prerequisites

- Go 1.18 or higher
- AWS credentials configured with appropriate permissions
- Terraform (for parsing configurations)

### Install from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/driftdetector.git
cd driftdetector

# Install dependencies
go mod download

# Build and install
go install ./cmd/driftdetector
```

### Using Make (recommended)

```bash
# Build the binary
make build

# Or install directly to $GOPATH/bin
make install
```

### Verifying Installation

```bash
driftdetector version
```

This should display the version information if installed correctly.

## 📖 CLI Usage

### Core Commands

| Command   | Description                                      |
|-----------|--------------------------------------------------|
| `detect`  | Check for configuration drift in EC2 instances  |
| `list`    | List EC2 instances managed by Terraform         |
| `version` | Show version information                        |

### List Command

List all EC2 instances that are managed by Terraform configurations.

#### Basic Usage

```bash
# List instances from a Terraform state file
driftdetector list --tf-state terraform.tfstate

# List instances from a Terraform directory
driftdetector list --tf-dir /path/to/terraform
```

#### Options

| Flag               | Description                                      | Required |
|--------------------|--------------------------------------------------|----------|
| `-s, --tf-state`   | Path to Terraform state file                     | Either   |
| `-d, --tf-dir`     | Path to Terraform configuration directory        | Either   |
| `-v, --verbose`    | Enable verbose output                            | No       |
| `-h, --help`       | Show help message                                | No       |

#### Examples

```bash
# Basic usage with state file
driftdetector list --tf-state terraform.tfstate

# From a Terraform directory
driftdetector list --tf-dir /path/to/terraform

# Show additional details with verbose output
driftdetector list --tf-state terraform.tfstate --verbose
```

### Detect Command

Compare an EC2 instance's current configuration with its Terraform definition and report any drifts.

#### Basic Usage

```bash
# Basic usage with state file
driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate

# Using a Terraform directory instead of state file
driftdetector detect -i i-1234567890abcdef0 -d /path/to/terraform
```

#### Options

| Flag                     | Description                                      | Required |
|--------------------------|--------------------------------------------------|----------|
| `-i, --instance-id`      | AWS EC2 instance ID to check                     | Yes      |
| `-s, --tf-state`         | Path to Terraform state file                     | Either   |
| `-d, --tf-dir`           | Path to Terraform configuration directory        | Either   |
| `-r, --region`           | AWS region (default: from AWS config)            | No       |
| `-o, --output`           | Output format (text, json) (default: "text")    | No       |
| `-v, --verbose`          | Enable verbose logging                           | No       |
| `-h, --help`             | Show help message                                | No       |

#### Examples

```bash
# Basic usage with state file
driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate

# Using a Terraform directory
driftdetector detect -i i-1234567890abcdef0 -d /path/to/terraform

# Specify AWS region
driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate -r us-west-2

# Output in JSON format (for programmatic use)
driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate -o json

# Enable verbose logging for debugging
driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate --verbose
```

#### Output Format

The tool provides detailed drift information in the following format:

```
Instance: i-1234567890abcdef0
Status: DRIFT_DETECTED

Drift Details:
┌───────────────────────┬────────────────────┬────────────────────┬─────────────┐
│ ATTRIBUTE             │ EXPECTED           │ ACTUAL            │ DRIFT TYPE  │
├───────────────────────┼────────────────────┼────────────────────┼─────────────┤
│ instance_type         │ t2.micro           │ t2.small          │ MODIFIED    │
│ tags.Environment      │ production         │ dev               │ MODIFIED    │
│ security_groups[0]    │ sg-12345678        │ sg-87654321       │ MODIFIED    │
│ tags.CreatedBy        │ terraform          │ -                 │ DELETED     │
│ tags.Owner            │ -                  │ admin@example.com │ ADDED       │
└───────────────────────┴────────────────────┴────────────────────┴─────────────┘
```

### Version Command

Display version information:

```bash
driftdetector version
```

### Example Output

#### Text Output (Default)
```
=== Drift Detection Results ===

Field: instance_type
AWS Value: t3.large
Terraform Value: t2.medium
Status: MODIFIED

Field: tags.Environment
AWS Value: production
Terraform Value: staging
Status: MODIFIED
```

#### JSON Output (with `-o json`)
```json
{
  "instance_id": "i-1234567890abcdef0",
  "timestamp": "2025-07-03T14:30:00Z",
  "drifts": [
    {
      "field": "instance_type",
      "aws_value": "t3.large",
      "terraform_value": "t2.medium",
      "status": "MODIFIED"
    },
    {
      "field": "tags.Environment",
      "aws_value": "production",
      "terraform_value": "staging",
      "status": "MODIFIED"
    }
  ]
}
```

## 🔧 Command Reference

### Global Options

These options are available for all commands:

| Flag           | Description                                      | Default                  |
|----------------|--------------------------------------------------|--------------------------|
| `-h, --help`   | Show help for the command                        |                          |
| `-o, --output` | Output format: `text` or `json`                  | `text`                   |
| `-r, --region` | AWS region to use                                | `AWS_REGION` env var     |
| `-v, --verbose`| Enable verbose output for debugging              | `false`                  |

### `detect` Command

Check for configuration drift in EC2 instances.

**Usage:**
```bash
driftdetector detect [flags]
```

**Flags:**
| Flag                | Description                                      | Required |
|---------------------|--------------------------------------------------|----------|
| `-i, --instance`    | EC2 instance ID to check                         | Yes      |
| `-s, --tf-state`    | Path to Terraform state file                     | Either   |
| `-d, --tf-dir`      | Path to Terraform configuration directory        | Either   |

### `list` Command

List EC2 instances managed by Terraform.

**Usage:**
```bash
driftdetector list [flags]
```

**Flags:**
| Flag                | Description                                      | Required |
|---------------------|--------------------------------------------------|----------|
| `-s, --tf-state`    | Path to Terraform state file                     | Either   |
| `-d, --tf-dir`      | Path to Terraform configuration directory        | Either   |

### `version` Command

Show version information.

**Usage:**
```bash
driftdetector version
```

### Examples

1. **Basic Usage**:
   ```bash
   driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate
   ```

2. **Check Multiple Instances**:
   ```bash
   for instance in $(driftdetector list -s terraform.tfstate | awk 'NR>1 {print $1}'); do
     driftdetector detect -i $instance -s terraform.tfstate
   done
   ```

3. **Generate JSON Report**:
   ```bash
   driftdetector detect -i i-1234567890abcdef0 -s terraform.tfstate -o json > drift_report.json
   ```

## Design

The tool is structured into several packages:

- `aws/`: AWS client and instance configuration retrieval
- `terraform/`: Terraform state and configuration parsing
- `detector/`: Drift detection logic
- `report/`: Report generation and formatting
- `cmd/`: Command-line interface

### Flow

1. **Configuration Retrieval**:
   - Fetch current EC2 instance configuration from AWS
   - Parse Terraform state or configuration files

2. **Drift Detection**:
   - Compare AWS configuration with Terraform configuration
   - Identify added, modified, or removed attributes

3. **Reporting**:
   - Generate human-readable or machine-readable reports
   - Highlight differences and potential issues

## Sample Data

### Sample AWS EC2 Response

```json
{
  "InstanceId": "i-1234567890abcdef0",
  "InstanceType": "t2.micro",
  "ImageId": "ami-0c55b159cbfafe1f0",
  "VpcId": "vpc-123456",
  "SubnetId": "subnet-123456",
  "KeyName": "my-key-pair",
  "SecurityGroups": [
    {
      "GroupId": "sg-123456",
      "GroupName": "my-security-group"
    }
  ],
  "Tags": [
    {
      "Key": "Name",
      "Value": "test-instance"
    }
  ]
}
```

### Sample Terraform Configuration

```hcl
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  subnet_id     = "subnet-123456"
  key_name      = "my-key-pair"
  
  vpc_security_group_ids = ["sg-123456"]
  
  tags = {
    Name = "test-instance"
  }
}
```

## Future Enhancements

- Support for additional AWS resources (RDS, S3, etc.)
- Integration with CI/CD pipelines
- Automated remediation of drift
- Support for Terraform Cloud/Enterprise
- Web-based dashboard for monitoring drift

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
