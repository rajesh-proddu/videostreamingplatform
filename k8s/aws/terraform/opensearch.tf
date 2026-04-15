# ─── AWS OpenSearch (managed Elasticsearch replacement) ─────────────────────

resource "aws_opensearch_domain" "main" {
  domain_name    = "videostreamingplatform-${var.environment}"
  engine_version = "OpenSearch_2.11"

  cluster_config {
    instance_type          = var.opensearch_instance_type
    instance_count         = var.opensearch_instance_count
    zone_awareness_enabled = var.opensearch_instance_count > 1

    dynamic "zone_awareness_config" {
      for_each = var.opensearch_instance_count > 1 ? [1] : []
      content {
        availability_zone_count = min(var.opensearch_instance_count, 3)
      }
    }
  }

  ebs_options {
    ebs_enabled = true
    volume_size = var.opensearch_volume_size
    volume_type = "gp3"
  }

  encrypt_at_rest {
    enabled = true
  }

  node_to_node_encryption {
    enabled = true
  }

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }

  vpc_options {
    subnet_ids         = var.opensearch_instance_count > 1 ? slice(aws_subnet.private[*].id, 0, min(var.opensearch_instance_count, 3)) : [aws_subnet.private[0].id]
    security_group_ids = [aws_security_group.opensearch.id]
  }

  access_policies = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "*" }
      Action    = "es:*"
      Resource  = "arn:aws:es:${var.aws_region}:${data.aws_caller_identity.current.account_id}:domain/videostreamingplatform-${var.environment}/*"
      Condition = {
        StringEquals = {
          "aws:PrincipalArn" = [
            aws_iam_role.opensearch_irsa.arn,
            aws_iam_role.eks_node_role.arn
          ]
        }
      }
    }]
  })

  tags = {
    Name = "videostreamingplatform-opensearch"
  }
}

# ─── Security Group for OpenSearch ──────────────────────────────────────────

resource "aws_security_group" "opensearch" {
  name        = "videostreamingplatform-opensearch-${var.environment}"
  description = "Security group for OpenSearch domain"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_group_ids = [aws_security_group.eks_nodes.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "opensearch-sg"
  }
}

# ─── IRSA: OpenSearch access for analytics + recommendations ────────────────

resource "aws_iam_role" "opensearch_irsa" {
  name = "opensearch-irsa-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = local.oidc_provider_arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${local.oidc_provider}:aud" = "sts.amazonaws.com"
            "${local.oidc_provider}:sub" = [
              "system:serviceaccount:analytics:kafka-es-consumer",
              "system:serviceaccount:recommendations:recommendations-sa"
            ]
          }
        }
      }
    ]
  })

  tags = {
    Name = "opensearch-irsa"
  }
}

resource "aws_iam_role_policy" "opensearch_access" {
  name = "opensearch-access"
  role = aws_iam_role.opensearch_irsa.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "es:ESHttpGet",
          "es:ESHttpPost",
          "es:ESHttpPut",
          "es:ESHttpDelete",
          "es:ESHttpHead"
        ]
        Resource = "${aws_opensearch_domain.main.arn}/*"
      }
    ]
  })
}
