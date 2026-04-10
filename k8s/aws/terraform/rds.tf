resource "aws_security_group" "rds" {
  name        = "videostreamingplatform-rds-${var.environment}"
  description = "Security group for RDS"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 3306
    to_port         = 3306
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
    Name = "rds-sg"
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "videostreamingplatform-db-subnet-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "db-subnet-group"
  }
}

resource "aws_rds_cluster" "main" {
  cluster_identifier              = "videostreamingplatform-${var.environment}"
  engine                          = "aurora-mysql"
  engine_version                  = "8.0.mysql_aurora.3.02.0"
  database_name                   = "videoplatform"
  master_username                 = var.rds_master_username
  master_password                 = var.rds_master_password
  db_subnet_group_name            = aws_db_subnet_group.main.name
  vpc_security_group_ids          = [aws_security_group.rds.id]
  backup_retention_period         = var.environment == "dev" ? 1 : 7
  skip_final_snapshot             = var.environment == "dev" ? true : false
  final_snapshot_identifier       = var.environment == "dev" ? null : "videostreamingplatform-final-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  enabled_cloudwatch_logs_exports = ["error", "slowquery"]
  storage_encrypted               = true
  kms_key_id                      = aws_kms_key.rds.arn
  deletion_protection             = var.environment == "prod" ? true : false

  tags = {
    Name = "rds-cluster"
  }
}

resource "aws_rds_cluster_instance" "main" {
  count              = var.rds_instance_count
  cluster_identifier = aws_rds_cluster.main.id
  instance_class     = var.rds_instance_class
  engine              = aws_rds_cluster.main.engine
  engine_version      = aws_rds_cluster.main.engine_version
  publicly_accessible = false

  tags = {
    Name = "rds-instance-${count.index + 1}"
  }
}

resource "aws_kms_key" "rds" {
  description             = "KMS key for RDS encryption"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "rds-encryption-key"
  }
}

resource "aws_kms_alias" "rds" {
  name          = "alias/videostreamingplatform-rds-${var.environment}"
  target_key_id = aws_kms_key.rds.key_id
}
