terraform {
  required_version = ">= 1.9"

  cloud {
    organization = "dreamshake"
    workspaces {
      name = "bet-with-goodall"
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
