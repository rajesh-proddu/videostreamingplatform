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

output "rds_endpoint" {
  description = "RDS instance endpoint"
  value       = aws_db_instance.main.endpoint
}

output "rds_address" {
  description = "RDS instance address (hostname only)"
  value       = aws_db_instance.main.address
}

output "rds_database_name" {
  description = "RDS database name"
  value       = aws_db_instance.main.db_name
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

output "metadata_service_irsa_role_arn" {
  description = "IRSA role ARN for metadata-service"
  value       = aws_iam_role.metadata_service_irsa.arn
}

output "data_service_irsa_role_arn" {
  description = "IRSA role ARN for data-service"
  value       = aws_iam_role.data_service_irsa.arn
}

output "eks_oidc_provider_arn" {
  description = "EKS OIDC provider ARN"
  value       = aws_iam_openid_connect_provider.eks.arn
}

output "opensearch_endpoint" {
  description = "OpenSearch domain endpoint"
  value       = "https://${aws_opensearch_domain.main.endpoint}"
}

output "opensearch_irsa_role_arn" {
  description = "IRSA role ARN for OpenSearch access"
  value       = aws_iam_role.opensearch_irsa.arn
}
