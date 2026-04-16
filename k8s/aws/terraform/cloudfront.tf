# ─── CloudFront CDN for video downloads ─────────────────────────────────────
# Sits in front of the S3 videos bucket to cache content at edge locations,
# reducing origin load and improving download latency globally.

resource "aws_cloudfront_origin_access_identity" "videos" {
  comment = "OAI for video streaming CDN"
}

# Allow CloudFront to read from the S3 bucket
resource "aws_s3_bucket_policy" "videos_cdn" {
  bucket = aws_s3_bucket.videos.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCloudFrontRead"
        Effect    = "Allow"
        Principal = {
          AWS = aws_cloudfront_origin_access_identity.videos.iam_arn
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.videos.arn}/*"
      }
    ]
  })
}

resource "aws_cloudfront_distribution" "videos" {
  enabled             = true
  comment             = "Video streaming CDN - ${var.environment}"
  default_root_object = ""
  price_class         = var.cloudfront_price_class

  origin {
    domain_name = aws_s3_bucket.videos.bucket_regional_domain_name
    origin_id   = "S3-videos"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.videos.cloudfront_access_identity_path
    }
  }

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-videos"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = var.cloudfront_default_ttl
    max_ttl                = var.cloudfront_max_ttl
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = {
    Name        = "videos-cdn-${var.environment}"
    Environment = var.environment
  }
}
