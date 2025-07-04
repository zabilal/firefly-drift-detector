provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  
  tags = {
    Name = "main-vpc"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
  
  tags = {
    Name = "main-subnet"
  }
}

resource "aws_security_group" "instance_sg" {
  name        = "instance-security-group"
  description = "Security group for the EC2 instance"
  vpc_id      = aws_vpc.main.id
  
  ingress {
    from_port   = 22
    to_port     = 22
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
    Name = "my-security-group"
  }
}

resource "aws_key_pair" "deployer" {
  key_name   = "my-key-pair"
  public_key = "ssh-rsa AAAAB3NzaC1yc2E..." # Replace with your public key
}

resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  subnet_id     = aws_subnet.main.id
  key_name      = aws_key_pair.deployer.key_name
  
  vpc_security_group_ids = [aws_security_group.instance_sg.id]
  
  root_block_device {
    volume_size = 8
    volume_type = "gp2"
  }
  
  tags = {
    Name        = "test-instance"
    Environment = "production"
    ManagedBy   = "terraform"
  }
}

output "instance_id" {
  value = aws_instance.example.id
}

output "public_ip" {
  value = aws_instance.example.public_ip
}
