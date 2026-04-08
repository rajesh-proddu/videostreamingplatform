environment = "dev"
aws_region  = "us-east-1"

vpc_cidr                 = "10.0.0.0/16"
public_subnet_cidrs      = ["10.0.1.0/24", "10.0.2.0/24"]
private_subnet_cidrs     = ["10.0.11.0/24", "10.0.12.0/24"]
kubernetes_version       = "1.28"
node_group_min_size      = 1
node_group_max_size      = 3
node_group_desired_size  = 2
node_instance_types      = ["t3.small"]
rds_instance_count       = 1
rds_instance_class       = "db.t3.small"
s3_video_retention_days  = 30
