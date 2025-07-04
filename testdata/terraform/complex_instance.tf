# Test file with a complex EC2 instance configuration

# Variables with defaults
variable "instance_type" {
  type    = string
  default = "t3.medium"
}

variable "ami_id" {
  type    = string
  default = "ami-0c55b159cbfafe1f0"
}

variable "environment" {
  type    = string
  default = "test"
}

# Data source example (not currently processed, but shouldn't cause errors)
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }
}

# Main EC2 instance resource
resource "aws_instance" "web" {
  ami           = var.ami_id
  instance_type = var.instance_type
  key_name      = "my-key-pair"

  tags = {
    Name        = "${var.environment}-web-server"
    Environment = var.environment
    ManagedBy   = "terraform"
  }

  # Root block device
  root_block_device {
    volume_size = 30
    volume_type = "gp3"
    encrypted   = true
    
    tags = {
      Name = "${var.environment}-root-volume"
    }
  }

  # EBS block device
  ebs_block_device {
    device_name = "/dev/sdh"
    volume_size = 50
    volume_type = "gp3"
    encrypted   = true
    iops        = 3000
    throughput  = 150
    
    tags = {
      Name = "${var.environment}-data-volume"
    }
  }

  # Network interface
  network_interface {
    device_index         = 0
    network_interface_id = aws_network_interface.web.id
  }

  # Metadata options
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
    instance_metadata_tags      = "enabled"
  }

  # Credit specification
  credit_specification {
    cpu_credits = "standard"
  }

  # User data (base64 encoded)
  user_data_base64 = base64encode(<<-EOF
              #!/bin/bash
              echo "Hello, World!" > /tmp/hello.txt
              EOF
  )
}

# Network interface (referenced by the instance)
resource "aws_network_interface" "web" {
  subnet_id = "subnet-12345678"
  
  tags = {
    Name = "${var.environment}-web-nic"
  }
}

# Security group
resource "aws_security_group" "web" {
  name_prefix = "${var.environment}-web-sg-"
  description = "Security group for web server"
  
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = {
    Name = "${var.environment}-web-sg"
  }
}

# Security group attachment (not processed, but shouldn't cause errors)
resource "aws_network_interface_sg_attachment" "web" {
  security_group_id    = aws_security_group.web.id
  network_interface_id = aws_network_interface.web.id
}
