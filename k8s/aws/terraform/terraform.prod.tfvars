environment = "prod"
aws_region  = "us-east-1"

vpc_cidr                 = "10.0.0.0/16"
public_subnet_cidrs      = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
private_subnet_cidrs     = ["10.0.11.0/24", "10.0.12.0/24", "10.0.13.0/24"]
kubernetes_version       = "1.28"
node_group_min_size      = 3
node_group_max_size      = 10
node_group_desired_size  = 5
node_instance_types      = ["t3.medium"]
rds_instance_count       = 3
rds_instance_class       = "db.t3.medium"
s3_video_retention_days  = 365
