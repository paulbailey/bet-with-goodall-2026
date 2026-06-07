output "site_bucket" {
  description = "S3 bucket name"
  value       = aws_s3_bucket.site.bucket
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID — use in deploy scripts to trigger invalidations"
  value       = aws_cloudfront_distribution.site.id
}

output "cloudfront_domain" {
  description = "CloudFront domain name (before DNS delegation)"
  value       = aws_cloudfront_distribution.site.domain_name
}

output "route53_zone_id" {
  description = "Route 53 hosted zone ID for betwithgoodall.com"
  value       = aws_route53_zone.site.zone_id
}

output "route53_nameservers" {
  description = "Point your domain registrar's NS records at these after first apply"
  value       = aws_route53_zone.site.name_servers
}

output "bet_builder_role_arn" {
  description = "IRSA role ARN — set this in builder/k8s/serviceaccount.yaml"
  value       = aws_iam_role.bet_builder.arn
}

output "fdb_api_key_parameter_name" {
  description = "SSM parameter to populate with the real football-data.org API key"
  value       = aws_ssm_parameter.fdb_api_key.name
}

output "anthropic_api_key_parameter_name" {
  description = "SSM parameter to populate with the real Anthropic API key (optional; daily-summary recap)"
  value       = aws_ssm_parameter.anthropic_api_key.name
}

output "frontend_deployer_role_arn" {
  description = "Set this as the AWS_ROLE_ARN variable in the GitHub repo Actions settings"
  value       = aws_iam_role.frontend_deployer.arn
}

output "push_subscriptions_table" {
  description = "DynamoDB table the builder reads to send Web Push notifications (set as PUSH_TABLE)"
  value       = aws_dynamodb_table.push_subscriptions.name
}

output "push_api_endpoint" {
  description = "Direct API Gateway endpoint for the push-subscription API (also fronted same-origin at /api/* via CloudFront)"
  value       = aws_apigatewayv2_api.push.api_endpoint
}

output "vapid_public_parameter_name" {
  description = "SSM parameter to populate with the VAPID public key (also set as the VITE_VAPID_PUBLIC_KEY Actions variable)"
  value       = aws_ssm_parameter.vapid_public.name
}

output "vapid_private_parameter_name" {
  description = "SSM parameter to populate with the VAPID private key"
  value       = aws_ssm_parameter.vapid_private.name
}
