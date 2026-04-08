resource "aws_s3_bucket" "videos" {
  bucket = "videostreamingplatform-videos-${var.environment}-${data.aws_caller_identity.current.account_id}"

  tags = {
    Name = "videos-bucket"
  }
}

resource "aws_s3_bucket_versioning" "videos" {
  bucket = aws_s3_bucket.videos.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "videos" {
  bucket = aws_s3_bucket.videos.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "videos" {
  bucket = aws_s3_bucket.videos.id

  rule {
    id     = "delete-old-videos"
    status = "Enabled"

    expiration {
      days = var.s3_video_retention_days
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

resource "aws_s3_bucket_public_access_block" "videos" {
  bucket = aws_s3_bucket.videos.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_kms_key" "s3" {
  description             = "KMS key for S3 encryption"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "s3-encryption-key"
  }
}

resource "aws_kms_alias" "s3" {
  name          = "alias/videostreamingplatform-s3-${var.environment}"
  target_key_id = aws_kms_key.s3.key_id
}

resource "aws_ecr_repository" "metadata_service" {
  name                 = "videostreamingplatform/metadata-service"
  image_tag_mutability = "MUTABLE"
  force_delete         = var.environment == "dev" ? true : false

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }

  tags = {
    Name = "metadata-service-repo"
  }
}

resource "aws_ecr_repository" "data_service" {
  name                 = "videostreamingplatform/data-service"
  image_tag_mutability = "MUTABLE"
  force_delete         = var.environment == "dev" ? true : false

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }

  tags = {
    Name = "data-service-repo"
  }
}

resource "aws_ecr_lifecycle_policy" "services" {
  repository = aws_ecr_repository.metadata_service.name

  lifecycle_policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 10 images"
      selection = {
        tagStatus     = "any"
        countType     = "imageCountMoreThan"
        countNumber   = 10
      }
      action = {
        type = "expire"
      }
    }]
  })
}

resource "aws_kms_key" "ecr" {
  description             = "KMS key for ECR encryption"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "ecr-encryption-key"
  }
}

resource "aws_kms_alias" "ecr" {
  name          = "alias/videostreamingplatform-ecr-${var.environment}"
  target_key_id = aws_kms_key.ecr.key_id
}

data "aws_caller_identity" "current" {}
