# ─── ElastiCache Redis for caching + rate limiting ──────────────────────────

resource "aws_elasticache_subnet_group" "main" {
  name       = "videostreamingplatform-redis-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "redis-subnet-group"
  }
}

resource "aws_security_group" "redis" {
  name        = "videostreamingplatform-redis-${var.environment}"
  description = "Security group for ElastiCache Redis"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 6379
    to_port         = 6379
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
    Name = "redis-sg"
  }
}

resource "aws_elasticache_replication_group" "main" {
  replication_group_id = "vsp-redis-${var.environment}"
  description          = "Redis for video streaming platform - ${var.environment}"

  engine               = "redis"
  engine_version       = "7.1"
  node_type            = var.redis_node_type
  num_cache_clusters   = var.environment == "prod" ? 2 : 1
  port                 = 6379
  parameter_group_name = "default.redis7"

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = false
  automatic_failover_enabled = var.environment == "prod" ? true : false

  snapshot_retention_limit = var.environment == "prod" ? 3 : 0

  tags = {
    Name = "videostreamingplatform-redis"
  }
}
