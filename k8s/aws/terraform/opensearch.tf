# ─── AWS OpenSearch ──────────────────────────────────────────────────────────
# OpenSearch domain creation requires service subscription.
# Enable the OpenSearch service in your AWS account and uncomment the domain
# resource below to provision it. Until then, ELASTICSEARCH_URL will be empty
# and the analytics consumer will be disabled.

# resource "aws_opensearch_domain" "main" { ... }

# ─── Security Group for OpenSearch (pre-created for future use) ────────────

resource "aws_security_group" "opensearch" {
  name        = "videostreamingplatform-opensearch-${var.environment}"
  description = "Security group for OpenSearch domain"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
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
