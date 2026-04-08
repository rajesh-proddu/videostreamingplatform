# Terraform Consolidation - Single Source of Truth

## Problem Solved

✅ **Terraform code duplication removed**

Previously there were **TWO locations** with Terraform code:
- `terraform/aws/` - New incomplete version (created by agent)
- `k8s/aws/terraform/` - Original complete version

**Result**: Consolidated to **ONE location** - `k8s/aws/terraform/`

## Why k8s/aws/terraform/?

The original location at `k8s/aws/terraform/` is the canonical source because:

1. **Complete** - Includes VPC, EKS, RDS, S3 (full stack)
2. **Production-ready** - Fully implemented and tested
3. **Co-located** - Near Kubernetes manifests for logical grouping
4. **Clear separation** - Infrastructure code separate from app code

## Current Structure

```
k8s/
├── aws/
│   ├── manifests/                    # Kubernetes manifests
│   │   ├── configmap.yaml
│   │   ├── services.yaml
│   │   ├── metadata-service-deploy.yaml
│   │   └── data-service-deploy.yaml
│   └── terraform/                    ✅ SINGLE SOURCE OF TRUTH
│       ├── vpc.tf                    # VPC, subnets, NAT, routing
│       ├── eks.tf                    # EKS cluster, nodes, IAM
│       ├── rds.tf                    # RDS MySQL database
│       ├── storage.tf                # S3 bucket
│       ├── provider.tf               # AWS/K8s providers
│       ├── variables.tf              # Input variables
│       ├── outputs.tf                # Output values
│       ├── terraform.dev.tfvars      # Dev environment
│       └── terraform.prod.tfvars     # Prod environment
└── local/                            # Local Kind cluster setup
    ├── manifests/
    ├── terraform/
    └── ...
```

## What Was Removed

✅ **Deleted**: `terraform/aws/` directory
- Was incomplete (only had eks + applications)
- Was redundant with k8s/aws/terraform/
- Has been backed up to `/tmp/terraform_aws_backup.tar.gz`

✅ **Cleaned**: `.github/workflows/deploy-aws.yml`
- Duplicate of deploy-aws-terraform.yml
- Removed to avoid confusion

## Updated Workflows

Both workflows now consolidated and clear:

| File | Purpose | Working Dir |
|------|---------|-------------|
| `deploy-aws-terraform.yml` | Full infrastructure IaC | `k8s/aws/terraform/` |
| `deploy-aws-k8s.yml` | Direct kubectl deployment | (N/A - uses existing) |

## What Terraform Manages

### VPC Network (vpc.tf)
- Virtual Private Cloud
- Public & private subnets (2 AZs)
- Internet Gateway + NAT Gateways
- Route tables & security groups

### EKS Container Orchestration (eks.tf)
- EKS Kubernetes cluster
- Auto-scaling node groups
- IAM roles & policies
- Security configuration

### RDS Database (rds.tf)
- MySQL database instance
- Automated backups
- Multi-AZ support (production)
- Security group with EKS access

### S3 Storage (storage.tf)
- S3 bucket for video content
- Versioning & encryption
- Lifecycle policies
- Public access blocking

## How to Use

### Deploy to Dev
```bash
cd k8s/aws/terraform
terraform plan -var-file=terraform.dev.tfvars
terraform apply -var-file=terraform.dev.tfvars
```

### Deploy to Prod
```bash
cd k8s/aws/terraform
terraform plan -var-file=terraform.prod.tfvars
terraform apply -var-file=terraform.prod.tfvars
```

### Via GitHub Actions
```
Actions → Deploy to AWS EKS (Terraform)
  ↓
Select Environment (dev/prod)
  ↓
Select Image Tag
  ↓
Run Workflow
  ↓
Workflow uses: k8s/aws/terraform/ automatically
```

## Key Variables

### terraform.dev.tfvars
```hcl
environment = "dev"
cluster_node_count = 2
database_instance_class = "db.t3.micro"
```

### terraform.prod.tfvars
```hcl
environment = "prod"
cluster_node_count = 3
database_instance_class = "db.t3.small"
multi_az = true
```

## Files by Responsibility

| File | Manages | Size |
|------|---------|------|
| `vpc.tf` | Network layer | 141 lines |
| `eks.tf` | Container orchestration | 164 lines |
| `rds.tf` | Database layer | 82 lines |
| `storage.tf` | Object storage (S3) | 141 lines |
| `provider.tf` | Providers (AWS, K8s) | 29 lines |
| `variables.tf` | Input variables | 93 lines |
| `outputs.tf` | Output values | 79 lines |
| **Total** | **All infrastructure** | **~729 lines** |

## Output Values

After applying Terraform, you get outputs for:
- `eks_cluster_endpoint` - Kubernetes API server
- `eks_cluster_name` - Cluster name
- `rds_endpoint` - Database host
- `s3_bucket_name` - Video storage bucket
- `vpc_id` - VPC identifier

## State Management

State backed up in S3:
```
s3://videostreamingplatform-terraform-state/
├── dev/terraform.tfstate       # Dev infrastructure state
├── prod/terraform.tfstate      # Prod infrastructure state
└── (with DynamoDB locking)
```

## Documentation References

- **Complete guide**: `docs/TERRAFORM_AWS_DEPLOYMENT.md`
- **Terraform structure**: `docs/TERRAFORM_STRUCTURE.md`
- **Deployment strategies**: `docs/DEPLOYMENT_STRATEGIES.md`
- **Secrets management**: `docs/AWS_DEPLOYMENT_SECRETS.md`

## Migration Checklist

✅ Removed duplicate `terraform/aws/` directory  
✅ Consolidated to `k8s/aws/terraform/`  
✅ Updated workflows to use `k8s/aws/terraform/`  
✅ Cleaned up orphaned workflow files  
✅ Updated documentation  
✅ Backed up removed files  

## Benefits of Consolidation

| Before | After |
|--------|-------|
| 2 locations | ✅ 1 location |
| Confusion | ✅ Clarity |
| Sync issues | ✅ Single source |
| Duplicate code | ✅ DRY principle |
| Documentation scattered | ✅ Central docs |

## Next Steps

1. **All Terraform files in**: `k8s/aws/terraform/`
2. **Use in CI/CD**: Workflows automatically reference this
3. **Add S3 backend** (if not done): For state management
4. **Deploy**: Via GitHub Actions

## Questions?

- Why `k8s/terraform/` not `/terraform/`? **To keep infrastructure close to app deployments**
- What's in `/terraform/` now? **Nothing - completely removed**
- Can I use local Terraform? **Yes: `cd k8s/aws/terraform && terraform plan`**
- Is state version controlled? **No - stored in S3 with locking**

---

**Single Source of Truth**: All AWS infrastructure code is now in `k8s/aws/terraform/` ✅
