terraform {
  required_version = ">= 1.9"

  cloud {
    organization = "dreamshake"
    workspaces {
      # Must match the name in the terraform-cloud-role trust policy in account_foundation
      name = "bet-with-goodall-2026"
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0, < 7.0"
    }
  }
}

# Primary region: eu-west-2 (London)
provider "aws" {
  region = "eu-west-2"

  default_tags {
    tags = {
      billing = "bet-with-goodall"
      project = "bet-with-goodall"
    }
  }
}

# CloudFront ACM certificates must be in us-east-1
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"

  default_tags {
    tags = {
      billing = "bet-with-goodall"
      project = "bet-with-goodall"
    }
  }
}
