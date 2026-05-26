# Route 53 Registrar auto-creates this zone when the domain is registered.
# The import block tells Terraform to adopt the existing zone rather than
# create a new one. Run `terraform apply` once after registration to import it;
# subsequent plans manage it normally.
#
# To find the zone ID after registration:
#   aws route53 list-hosted-zones-by-name --dns-name betwithgoodall.com \
#     --query 'HostedZones[0].Id' --output text --region eu-west-2
import {
  to = aws_route53_zone.site
  id = "Z07141303RQOONNKGRDB8" # paste the zone ID here (format: Z0123456789ABCDEFGHIJ)
}

resource "aws_route53_zone" "site" {
  name = "betwithgoodall.com"
}

# Root domain A + AAAA → CloudFront
resource "aws_route53_record" "root_a" {
  zone_id = aws_route53_zone.site.zone_id
  name    = "betwithgoodall.com"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.site.domain_name
    zone_id                = aws_cloudfront_distribution.site.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "root_aaaa" {
  zone_id = aws_route53_zone.site.zone_id
  name    = "betwithgoodall.com"
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.site.domain_name
    zone_id                = aws_cloudfront_distribution.site.hosted_zone_id
    evaluate_target_health = false
  }
}

# www subdomain A + AAAA → CloudFront
resource "aws_route53_record" "www_a" {
  zone_id = aws_route53_zone.site.zone_id
  name    = "www.betwithgoodall.com"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.site.domain_name
    zone_id                = aws_cloudfront_distribution.site.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "www_aaaa" {
  zone_id = aws_route53_zone.site.zone_id
  name    = "www.betwithgoodall.com"
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.site.domain_name
    zone_id                = aws_cloudfront_distribution.site.hosted_zone_id
    evaluate_target_health = false
  }
}
