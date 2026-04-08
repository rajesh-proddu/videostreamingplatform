output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = aws_eks_cluster.main.name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = aws_eks_cluster.main.endpoint
}

output "eks_cluster_version" {
  description = "EKS cluster version"
  value       = aws_eks_cluster.main.version
}

output "eks_cluster_arn" {
  description = "EKS cluster ARN"
  value       = aws_eks_cluster.main.arn
}

output "eks_node_group_id" {
  description = "EKS node group ID"
  value       = aws_eks_node_group.main.id
}

output "rds_cluster_endpoint" {
  description = "RDS cluster endpoint"
  value       = aws_rds_cluster.main.endpoint
}

output "rds_reader_endpoint" {
  description = "RDS cluster reader endpoint"
  value       = aws_rds_cluster.main.reader_endpoint
}

output "rds_database_name" {
  description = "RDS database name"
  value       = aws_rds_cluster.main.database_name
}

output "s3_videos_bucket" {
  description = "S3 bucket for video storage"
  value       = aws_s3_bucket.videos.bucket
}

output "s3_videos_bucket_arn" {
  description = "S3 bucket ARN"
  value       = aws_s3_bucket.videos.arn
}

output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = aws_subnet.private[*].id
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = aws_subnet.public[*].id
}

output "eks_node_group_status" {
  description = "EKS node group status"
  value       = aws_eks_node_group.main.status
}
