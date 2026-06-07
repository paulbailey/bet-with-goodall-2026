# Web Push notifications: a subscription registry the static site writes to and
# the builder reads from to send match-result pushes.
#
#   browser ──POST /api/subscribe──▶ CloudFront ──▶ Lambda (function URL) ──▶ DynamoDB
#   builder ──Scan/Delete───────────────────────────────────────────────────▶ DynamoDB
#
# The /api/* CloudFront behaviour and origin are added in cloudfront.tf so the
# endpoint is same-origin (no CORS) with the rest of the site.

# ── Subscription store ────────────────────────────────────────────────────────

resource "aws_dynamodb_table" "push_subscriptions" {
  name         = "bet-with-goodall-push-subscriptions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id" # sha256(endpoint)

  attribute {
    name = "id"
    type = "S"
  }
}

# ── Subscription API Lambda ─────────────────────────────────────────────────--

data "archive_file" "subscriptions_lambda" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/subscriptions"
  output_path = "${path.module}/.build/subscriptions-lambda.zip"
}

data "aws_iam_policy_document" "subscriptions_lambda_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "subscriptions_lambda" {
  name               = "bet-with-goodall-subscriptions-lambda"
  description        = "Execution role for the push-subscription API Lambda"
  assume_role_policy = data.aws_iam_policy_document.subscriptions_lambda_assume.json
}

resource "aws_iam_role_policy_attachment" "subscriptions_lambda_logs" {
  role       = aws_iam_role.subscriptions_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "subscriptions_lambda_ddb" {
  name = "ddb-write"
  role = aws_iam_role.subscriptions_lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:DeleteItem"]
        Resource = aws_dynamodb_table.push_subscriptions.arn
      }
    ]
  })
}

resource "aws_lambda_function" "subscriptions" {
  function_name    = "bet-with-goodall-subscriptions"
  role             = aws_iam_role.subscriptions_lambda.arn
  runtime          = "nodejs22.x"
  handler          = "index.handler"
  filename         = data.archive_file.subscriptions_lambda.output_path
  source_code_hash = data.archive_file.subscriptions_lambda.output_base64sha256
  timeout          = 10

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.push_subscriptions.name
    }
  }
}

# ── Lambda function URL in front of the Lambda ───────────────────────────────-
# A plain function URL — no API Gateway. CloudFront fronts it same-origin at
# /api/* (see cloudfront.tf), so there's no CORS to manage and the routing
# (/api/subscribe vs /api/unsubscribe) is done in the handler. AuthType NONE +
# the public invoke permission make it reachable; it was effectively public
# behind API Gateway too.

resource "aws_lambda_function_url" "subscriptions" {
  function_name      = aws_lambda_function.subscriptions.function_name
  authorization_type = "NONE"
}

resource "aws_lambda_permission" "function_url" {
  statement_id           = "AllowPublicFunctionUrl"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.subscriptions.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}

# ── VAPID keys (Web Push signing) ──────────────────────────────────────────---
# Generate a keypair once (see builder/README.md → Push notifications), then
# populate these parameters. The public key is also set as the
# VITE_VAPID_PUBLIC_KEY GitHub Actions variable for the frontend build.
# Same placeholder pattern as the other secrets: created here, value filled in
# afterwards and never overwritten by Terraform.

resource "aws_ssm_parameter" "vapid_public" {
  name        = "/homelab/bet-with-goodall/builder/vapid_public"
  description = "Web Push VAPID public key (non-secret; also set as VITE_VAPID_PUBLIC_KEY in Actions)"
  type        = "String"
  value       = "REPLACE_ME_IN_AWS_PARAMETER_STORE"

  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ssm_parameter" "vapid_private" {
  name        = "/homelab/bet-with-goodall/builder/vapid_private"
  description = "Web Push VAPID private key for the bet-with-goodall builder"
  type        = "SecureString"
  value       = "REPLACE_ME_IN_AWS_PARAMETER_STORE"

  lifecycle {
    ignore_changes = [value]
  }
}

# ── Builder access to the subscription table ───────────────────────────────---
# The builder scans the table to send pushes and prunes entries that the push
# service reports as gone (HTTP 404/410).

resource "aws_iam_role_policy" "bet_builder_push" {
  name = "bet-builder-push"
  role = aws_iam_role.bet_builder.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadPruneSubscriptions"
        Effect = "Allow"
        Action = [
          "dynamodb:Scan",
          "dynamodb:DeleteItem",
        ]
        Resource = aws_dynamodb_table.push_subscriptions.arn
      }
    ]
  })
}
