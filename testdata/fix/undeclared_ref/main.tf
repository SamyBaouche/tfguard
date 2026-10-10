terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = "ca-central-1"
}

resource "aws_s3_bucket" "logs" {
  bucket = "tfguard-demo-logs-${var.environment}"

  tags = {
    Environment = var.environment
  }
}
