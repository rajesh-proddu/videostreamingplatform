# Terraform Configuration Structure

## Single Source of Truth

All Terraform infrastructure code is located in **one place**:

```
k8s/aws/terraform/
├── vpc.tf                    # VPC, subnets, NAT gateways, routing
├── eks.tf                    # EKS cluster, node groups, IAM roles
├── rds.tf                    # RDS MySQL database
├── storage.tf                # S3 bucket for video storage
├── provider.tf               # AWS, Kubernetes providers
├── variables.tf              # Input variables
├── outputs.tf                # Output values (endpoints, IDs)
├── terraform.dev.tfvars      # Development environment values
└── terraform.prod.tfvars     # Production environment values
```

## Why Here?

The `k8s/aws/terraform/` location has several advantages:

1. **Co-located with Kubernetes manifests** - Both infrastructure and app deployments in one logical place
2. **Complete AWS setup** - Includes VPC, EKS, RDS, S3 (full stack IaC)
3. **Clear organization** - Separate from application code
4. **Local deployments** - `k8s/local/` also nearby for comparison

## What's Managed

| Component | File | Status |
|-----------|------|--------|
| VPC & Networking | vpc.tf | ✅ Complete |
| EKS Cluster & Nodes | eks.tf | ✅ Complete |
| RDS Database | rds.tf | ✅ Complete |
| S3 Storage | storage.tf | ✅ Complete |
| AWS Providers | provider.tf | ✅ Complete |
| Variables | variables.tf | ✅ Complete |
| Outputs | outputs.tf | ✅ Complete |

## File Overview

### vpc.tf (Network Layer)
```
- Creates VPC with configurable CIDR block
- Public subnets across 2 AZs
- Private subnets across 2 AZs
- Internet Gateway for public access
- NAT Gateways for private subnet internet access
- Route tables for public and private routing
- Security groups for EKS and RDS
```

### eks.tf (Container Orchestration)
```
- EKS cluster with auto-scaling
- Node groups (worker nodes)
- IAM roles for cluster and nodes
- Security group for cluster communication
- Kubernetes provider configuration
```

### rds.tf (Database)
```
- MySQL RDS instance
- DB security group (only allows EKS pods)
- Database subnet group
- Backup and retention settings
- Multi-AZ support for production
```

### storage.tf (Object Storage)
```
- S3 bucket for video content
- Bucket versioning
- Encryption at rest
- Public access blocking
- Lifecycle policies for cost optimization
```

## Usage

### Local Testing

```bash
cd k8s/aws/terraform

# Initialize (one-time)
terraform init

# Plan changes
terraform plan -var-file=terraform.dev.tfvars

# Apply changes
terraform apply -var-file=terraform.dev.tfvars -var="database_password=xxx"

# Destroy (cleanup)
terraform destroy -var-file=terraform.dev.tfvars
```

### GitHub Actions Deployment

```bash
# Automatically uses k8s/aws/terraform/ directory
# Set working-directory: k8s/aws/terraform
# Points to terraform.dev.tfvars or terraform.prod.tfvars
```

## Environment Configuration

### Development (terraform.dev.tfvars)
```hcl
environment               = "dev"
cluster_node_count        = 2
database_instance_class   = "db.t3.micro"
enable_detailed_monitoring = false
```

### Production (terraform.prod.tfvars)
```hcl
environment               = "prod"
cluster_node_count        = 3
database_instance_class   = "db.t3.small"
enable_detailed_monitoring = true
multi_az                  = true
```

## Variables

Key input variables in `variables.tf`:

```hcl
variable "environment" {
  description = "Environment name (dev/prod)"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  default     = "10.0.0.0/16"
}

variable "database_password" {
  description = "RDS Master password"
  type        = string
  sensitive   = true
}

variable "container_image_tag" {
  description = "Docker image tag"
  default     = "latest"
}
```

## Outputs

Key output values in `outputs.tf`:

```hcl
output "eks_cluster_endpoint" {
  value = aws_eks_cluster.main.endpoint
}

output "eks_cluster_name" {
  value = aws_eks_cluster.main.name
}

output "rds_endpoint" {
  value = aws_db_instance.main.endpoint
}

output "s3_bucket_name" {
  value = aws_s3_bucket.video_storage.id
}

output "vpc_id" {
  value = aws_vpc.main.id
}
```

## State Management

### Backend Configuration
State is stored in **S3 with locking**:

```hcl
backend "s3" {
  bucket         = "videostreamingplatform-terraform-state"
  key            = "aws/terraform.tfstate"
  region         = "us-east-1"
  dynamodb_table = "terraform-locks"
  encrypt        = true
}
```

### State Files
```
s3://videostreamingplatform-terraform-state/
├── dev/terraform.tfstate       # Development state
├── dev/terraform.tfstate.backup
├── prod/terraform.tfstate      # Production state
└── prod/terraform.tfstate.backup
```

## Workflows

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy-aws-terraform.yml
working-directory: k8s/aws/terraform

terraform init -backend-config=...
terraform plan -var-file=terraform.${env}.tfvars
terraform apply tfplan
```

## Troubleshooting

### State Lock Issues
```bash
cd k8s/aws/terraform
terraform force-unlock LOCK_ID
```

### Terraform Validation
```bash
cd k8s/aws/terraform
terraform validate
terraform fmt -check
```

### View Resources
```bash
terraform state list
terraform state show aws_eks_cluster.main
terraform graph  # Visualize dependencies
```

## Best Practices

✅ **Commit tfvars files** - terraform.dev/prod.tfvars are safe (no secrets)  
❌ **Don't commit** - terraform.tfstate (use S3 backend)  
✅ **Use variables** - Pass secrets via GitHub Secrets  
✅ **Plan before apply** - Always review `terraform plan` output  
✅ **Lock DynamoDB** - Prevent concurrent modifications  
✅ **Version control** - All .tf files in git  

## Migration Path

```
Manual AWS Setup
     ↓
Terraform Plan Review
     ↓
Terraform Apply (create)
     ↓
Terraform Plan (updates)
     ↓
Terraform Apply (modify)
     ↓
Terraform Destroy (cleanup)
```

## Related Files

- **Kubernetes manifests**: `k8s/aws/manifests/` (deployed by Terraform)
- **Local development**: `k8s/local/terraform/` (Kind cluster)
- **Deployment workflows**: `.github/workflows/deploy-aws-terraform.yml`
- **Documentation**: `docs/TERRAFORM_AWS_DEPLOYMENT.md`

## Commands Reference

```bash
# Navigate to Terraform directory
cd k8s/aws/terraform

# Initialize workspace
terraform init

# Format code
terraform fmt -recursive

# Validate configuration
terraform validate

# Plan changes (dev)
terraform plan -var-file=terraform.dev.tfvars \
  -var="database_password=$DB_PW"

# Plan changes (prod)
terraform plan -var-file=terraform.prod.tfvars \
  -var="database_password=$DB_PW"

# Apply changes
terraform apply tfplan

# Destroy infrastructure
terraform destroy -var-file=terraform.prod.tfvars

# Show outputs
terraform output
terraform output eks_cluster_endpoint

# Debug
terraform console
terraform graph
```

---

**Summary**: All Terraform code is in `k8s/aws/terraform/` - a single, complete definition of the entire AWS infrastructure (VPC, EKS, RDS, S3). This is the source of truth for infrastructure as code.
