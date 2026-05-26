data "aws_caller_identity" "current" {}

# SSM parameter for the football-data.org API key.
# Terraform creates it with a placeholder; populate the real value in AWS
# console / CLI afterward (Terraform will not overwrite it on subsequent applies).
resource "aws_ssm_parameter" "fdb_api_key" {
  name        = "/homelab/bet-with-goodall/builder/fdb_api_key"
  description = "football-data.org API key for the bet-with-goodall builder"
  type        = "SecureString"
  value       = "REPLACE_ME_IN_AWS_PARAMETER_STORE"

  lifecycle {
    ignore_changes = [value]
  }
}

# Reference the existing OIDC provider created by the homelab-aws workspace.
# This is a data source — it reads the existing resource without touching it.
data "aws_iam_openid_connect_provider" "homelab" {
  url = "https://oidc.dreamshake.net"
}

locals {
  # Strip the scheme so we can use the hostpath as a condition key prefix
  oidc_issuer_hostpath = "oidc.dreamshake.net"
}

# Trust policy: allow the bet-builder ServiceAccount (default/bet-builder)
# to assume this role via IRSA
data "aws_iam_policy_document" "bet_builder_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.homelab.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer_hostpath}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer_hostpath}:sub"
      values   = ["system:serviceaccount:default:bet-builder"]
    }
  }
}

resource "aws_iam_role" "bet_builder" {
  name               = "bet-builder"
  description        = "IRSA role for the bet-with-goodall builder (k8s default/bet-builder SA)"
  assume_role_policy = data.aws_iam_policy_document.bet_builder_assume_role.json
}

# Minimal S3 permissions: list the bucket, read/write only state.json
resource "aws_iam_role_policy" "bet_builder_s3" {
  name = "bet-builder-s3"
  role = aws_iam_role.bet_builder.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ListBucket"
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.site.arn
      },
      {
        Sid    = "ReadWriteStateJson"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
        ]
        Resource = "${aws_s3_bucket.site.arn}/data/state.json"
      }
    ]
  })
}
