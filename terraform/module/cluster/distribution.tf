resource "aws_cloudfront_vpc_origin" "this" {
  vpc_origin_endpoint_config {
    arn                    = aws_lb.this.arn
    http_port              = 80
    https_port             = 443
    name                   = "cluster-${var.name}"
    origin_protocol_policy = "http-only"

    origin_ssl_protocols {
      items    = ["TLSv1.2"]
      quantity = 1
    }
  }
}

resource "aws_cloudfront_distribution" "this" {
  enabled     = true
  price_class = "PriceClass_100"

  origin {
    domain_name = aws_lb.this.dns_name
    origin_id   = "cluster-${var.name}"

    vpc_origin_config {
      vpc_origin_id = aws_cloudfront_vpc_origin.this.id
    }
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "cluster-${var.name}"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = true

      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
