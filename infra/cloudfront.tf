resource "aws_cloudfront_origin_access_control" "site" {
  name                              = "bet-with-goodall"
  description                       = "OAC for betwithgoodall.com S3 origin"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "site" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  price_class         = "PriceClass_100" # US, Canada, Europe

  aliases = ["betwithgoodall.com", "www.betwithgoodall.com"]

  origin {
    domain_name              = aws_s3_bucket.site.bucket_regional_domain_name
    origin_id                = "s3-betwithgoodall"
    origin_access_control_id = aws_cloudfront_origin_access_control.site.id
  }

  # Default: no caching — keeps index.html and data/state.json always fresh
  default_cache_behavior {
    target_origin_id       = "s3-betwithgoodall"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]

    # AWS managed: CachingDisabled
    cache_policy_id = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
  }

  # /assets/*: long cache — Vite content-hashes these filenames so they're safe to cache forever
  ordered_cache_behavior {
    path_pattern           = "/assets/*"
    target_origin_id       = "s3-betwithgoodall"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]

    # AWS managed: CachingOptimized (default TTL 86400s, max 31536000s)
    cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  }

  # SPA fallback: S3 returns 403 for unknown paths → serve index.html
  custom_error_response {
    error_code            = 403
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 10
  }

  custom_error_response {
    error_code            = 404
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 10
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.site.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}
