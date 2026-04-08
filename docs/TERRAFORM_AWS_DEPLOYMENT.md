# Terraform AWS Deployment

Your deployment workflow has been converted from direct `kubectl` commands to **Terraform-based Infrastructure as Code (IaC)**.

## Benefits

✅ **Version Control** - All infrastructure changes tracked in git  
✅ **Consistency** - Same infrastructure replicated across environments  
✅ **Rollback** - Easy to revert infrastructure changes  
✅ **State Management** - Terraform maintains state in S3 with locking  
✅ **Secrets Protection** - Sensitive data passed via GitHub Secrets  
✅ **Documentation** - Infrastructure readable and documented in code  

## Directory Structure

```
terraform/aws/
├── provider.tf           # AWS, Kubernetes, Helm providers
├── variables.tf          # Global variables and validation
├── outputs.tf            # Output values (endpoints, IDs)
├── eks/
│   └── main.tf          # VPC, EKS cluster, node groups
├── applications/
│   ├── main.tf          # Deployments, services, configmaps, secrets
│   └── variables.tf     # Application-specific variables
├── dev.tfvars           # Development environment values
└── prod.tfvars          # Production environment values
```

## What's Managed by Terraform

### Infrastructure (eks/main.tf)
- **VPC** - Virtual Private Cloud with public/private subnets across 2 AZs
- **Internet Gateway** - Public internet access
- **NAT Gateways** - Private subnet internet access
- **Route Tables** - Public and private routing
- **Security Groups** - Network access control
- **EKS Cluster** - Managed Kubernetes cluster (v1.27)
- **Node Groups** - Auto-scaling worker nodes
- **IAM Roles** - Cluster and node permissions

### Applications (applications/main.tf)
- **Kubernetes Namespace** - `videostreamingplatform`
- **Deployments**:
  - Metadata Service (1-3 replicas)
  - Data Service (1-3 replicas)
- **Services** - LoadBalancer type for both services
- **ConfigMaps** - Application configuration
- **Secrets** - Database passwords (encrypted in Kubernetes)

## Environment Configuration

### Development (dev.tfvars)
```hcl
environment              = "dev"
node_group_desired_size  = 2      # Fewer nodes in dev
metadata_service_replicas = 1     # Single replica
data_service_replicas    = 1
node_instance_types      = ["t3.medium"]
```

### Production (prod.tfvars)
```hcl
environment              = "prod"
node_group_desired_size  = 3      # More redundancy
metadata_service_replicas = 3     # High availability
data_service_replicas    = 3
node_instance_types      = ["t3.large"]
```

## GitHub Actions Workflow

Your updated `.github/workflows/deploy-aws.yml`:

1. **Checkout code** - Get Terraform files
2. **Configure AWS** - Use IAM role (OIDC)
3. **Setup Terraform** - Install v1.5.0
4. **Terraform Init** - Initialize S3 backend
5. **Terraform Plan** - Review changes (output: tfplan)
6. **Terraform Apply** - Create/update infrastructure
7. **Wait for Services** - kubectl rollout status
8. **Smoke Tests** - Test health endpoints
9. **Slack Notification** - Success/failure alert

## Setup Requirements

### 1. Create S3 Backend (One time)

```bash
# Create S3 bucket for state
aws s3api create-bucket \
  --bucket videostreamingplatform-terraform-state \
  --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket videostreamingplatform-terraform-state \
  --versioning-configuration Status=Enabled

# Enable encryption
aws s3api put-bucket-encryption \
  --bucket videostreamingplatform-terraform-state \
  --server-side-encryption-configuration '{
    "Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}}]
  }'

# Block public access
aws s3api put-public-access-block \
  --bucket videostreamingplatform-terraform-state \
  --public-access-block-configuration \
  "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true"

# Create DynamoDB for state locking
aws dynamodb create-table \
  --table-name terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

### 2. Add GitHub Secrets

In GitHub repo → Settings → Secrets and variables → Actions:

```
AWS_ROLE_ARN                    = arn:aws:iam::ACCOUNT_ID:role/GitHubActionsRole
TERRAFORM_STATE_BUCKET          = videostreamingplatform-terraform-state
TERRAFORM_LOCK_TABLE            = terraform-locks
DATABASE_HOST                   = your-rds-endpoint.rds.amazonaws.com
DATABASE_PASSWORD               = your-db-password
SLACK_WEBHOOK                   = https://hooks.slack.com/...
```

### 3. Set Environment Protection Rules

In GitHub repo → Settings → Environments → {dev, prod}:

- ✅ **Require reviewers** (for prod)
- ✅ **Restrict to main branch**
- ✅ **Add deployment reviewers** (team leads for prod)

## Local Development

### Prerequisites
```bash
brew install terraform aws-cli kubectl
# or on Linux
wget https://releases.hashicorp.com/terraform/1.5.0/terraform_1.5.0_linux_amd64.zip
unzip terraform_1.5.0_linux_amd64.zip && sudo mv terraform /usr/local/bin/
```

### Deploy Locally

```bash
cd k8s/aws/terraform

# Initialize (one time)
terraform init \
  -backend-config="bucket=videostreamingplatform-terraform-state" \
  -backend-config="key=dev/terraform.tfstate" \
  -backend-config="region=us-east-1" \
  -backend-config="dynamodb_table=terraform-locks" \
  -backend-config="encrypt=true"

# Plan changes (dry run)
terraform plan \
  -var-file="dev.tfvars" \
  -var="container_image_tag=latest" \
  -var="database_password=$DB_PASSWORD" \
  -var="database_host=$DB_HOST"

# Apply changes
terraform apply \
  -var-file="dev.tfvars" \
  -var="container_image_tag=latest" \
  -var="database_password=$DB_PASSWORD" \
  -var="database_host=$DB_HOST"

# See outputs
terraform output
terraform output metadata_service_endpoint
```

### Destroy Infrastructure (Cleanup)

```bash
terraform destroy \
  -var-file="dev.tfvars" \
  -var="container_image_tag=latest" \
  -var="database_password=$DB_PASSWORD" \
  -var="database_host=$DB_HOST"
```

## Terraform State Management

### View State
```bash
terraform state list                      # List all resources
terraform state show aws_eks_cluster.videostreamingplatform
terraform state pull                      # Get full state (S3 backend)
```

### State Locking
- Prevents concurrent modifications
- DynamoDB table: `terraform-locks`
- Automatic 10-minute timeout

### State File Location
```
s3://videostreamingplatform-terraform-state/
├── dev/terraform.tfstate        # Dev state
├── dev/terraform.tfstate.backup # Dev backup
├── prod/terraform.tfstate       # Prod state
└── prod/terraform.tfstate.backup
```

## Workflow Steps

### 1. Deploy to Dev
```bash
# GitHub Actions: Manually trigger workflow
# 1. Click "Actions" → "Deploy to AWS EKS (Terraform)"
# 2. Select environment: dev
# 3. Enter image_tag: v1.0.0
# 4. Click "Run workflow"
```

### 2. Review Changes
```bash
# GitHub shows terraform plan output
# Review resource changes before applying
# Approve or cancel workflow
```

### 3. Auto-Apply if Approved
```bash
# Terraform applies changes automatically
# Services rollout
# Smoke tests run
# Slack notification sent
```

### 4. Deploy to Prod (with Reviewers)
```bash
# Same steps but prod environment
# Requires manual approval from team leads
# Different secrets used (prod credentials)
```

## Common Operations

### Update Service Image
```bash
# Just change image_tag in GitHub Actions workflow input
# Terraform reads container_image_tag variable
# kubectl set image is automatic via terraform
```

### Scale Services
```bash
# Edit prod.tfvars or dev.tfvars
data_service_replicas = 5  # Change this
git commit -am "Scale data service"

# Trigger workflow - Terraform applies new replicas
```

### Add Environment Variable
```hcl
# In terraform/aws/applications/main.tf
env {
  name  = "NEW_VAR"
  value = "new_value"
}

# terraform plan shows change
# terraform apply creates it
```

### Destroy Infrastructure (Prod Warning ⚠️)
```bash
# GitHub Actions: Check "destroy" checkbox
# This runs: terraform destroy
# WARNING: Deletes all AWS resources (EKS, RDS, etc.)
```

## Troubleshooting

### Terraform State Lock Stuck
```bash
# Force unlock (use carefully!)
terraform force-unlock LOCK_ID
```

### State Mismatch
```bash
# Refresh state from actual resources
terraform refresh
terraform plan
```

### Cannot access S3 backend
```bash
# Verify AWS credentials
aws sts get-caller-identity
# Verify S3 bucket exists
aws s3 ls s3://videostreamingplatform-terraform-state/
```

## Security Best Practices

✅ **State encrypted** - S3 backend with AES256  
✅ **State locked** - DynamoDB prevents concurrent modifications  
✅ **No secrets in code** - Use GitHub Secrets (${{ secrets.VAR }})  
✅ **IAM role only** - No hardcoded AWS keys  
✅ **Approval required** - For production deployments  
✅ **Audit trail** - All changes in git history and Terraform state  

## Next Steps

1. **Create S3 backend** - Run setup commands above
2. **Add GitHub Secrets** - Configure AWS_ROLE_ARN, DATABASE_HOST, etc.
3. **Test workflow locally** - `terraform plan`
4. **Deploy dev** - Trigger workflow with dev environment
5. **Monitor state** - Check S3 backend and DynamoDB

## References

- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest)
- [Terraform Kubernetes Provider](https://registry.terraform.io/providers/hashicorp/kubernetes/latest)
- [AWS EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [Terraform State Management](https://www.terraform.io/language/state)
