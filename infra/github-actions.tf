# The token.actions.githubusercontent.com OIDC provider is already registered
# in this account by the mono/infra workspace. Reference it as a data source.
data "aws_iam_openid_connect_provider" "github_actions" {
  url = "https://token.actions.githubusercontent.com"
}

data "aws_iam_policy_document" "frontend_deployer_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github_actions.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Restrict to pushes/deployments on this repo only (any ref)
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:paulbailey/bet-with-goodall-2026:*"]
    }
  }
}

resource "aws_iam_role" "frontend_deployer" {
  name               = "bet-with-goodall-frontend-deployer"
  description        = "Assumed by GitHub Actions to deploy the bet-with-goodall frontend"
  assume_role_policy = data.aws_iam_policy_document.frontend_deployer_assume_role.json
}

resource "aws_iam_role_policy" "frontend_deployer" {
  name = "bet-with-goodall-frontend-deployer"
  role = aws_iam_role.frontend_deployer.id

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
        Sid    = "SyncFrontend"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:GetObject",
        ]
        Resource = "${aws_s3_bucket.site.arn}/*"
      },
      {
        Sid      = "InvalidateCloudFront"
        Effect   = "Allow"
        Action   = ["cloudfront:CreateInvalidation"]
        Resource = aws_cloudfront_distribution.site.arn
      },
    ]
  })
}
