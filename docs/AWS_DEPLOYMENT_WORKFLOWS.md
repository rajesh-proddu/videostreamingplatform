# AWS Deployment Workflows

Two deployment workflows available for different scenarios:

## 1. Terraform-Based Deployment (`deploy-aws-terraform.yml`)

**Purpose**: Full infrastructure provisioning and management

**What it does:**
- Creates VPC, subnets, NAT gateways, routing
- Provisions EKS cluster with auto-scaling node groups
- Deploys Kubernetes resources (deployments, services, configmaps, secrets)
- Manages complete infrastructure as code
- Provides state management with S3 backend and DynamoDB locking

**When to use:**
- ✅ Initial infrastructure setup
- ✅ Creating multiple environments (dev, prod)
- ✅ Infrastructure changes needed
- ✅ Want version-controlled infrastructure
- ✅ Need state management and rollback capability

**How to trigger:**
```
GitHub Actions → Deploy to AWS EKS (Terraform) → Select env & image tag
```

**Prerequisites:**
- S3 bucket for Terraform state
- DynamoDB table for state locking
- GitHub Secrets: AWS_ROLE_ARN, TERRAFORM_STATE_BUCKET, TERRAFORM_LOCK_TABLE, DATABASE_HOST
- RDS master password is now AWS-managed (Secrets Manager) — fetched at deploy time, no GitHub secret needed

**Deployment time:** 15-20 minutes

---

## 2. kubectl-Based Deployment (`deploy-aws-k8s.yml`)

**Purpose**: Application deployment to existing cluster

**What it does:**
- Configures kubectl with existing EKS cluster
- Applies Kubernetes manifests (services, deployments)
- Updates container image tags
- Performs health checks and smoke tests

**When to use:**
- ✅ Cluster already exists
- ✅ Quick application updates
- ✅ Updating only container images
- ✅ Simple deployments
- ✅ Emergency hotfixes

**How to trigger:**
```
GitHub Actions → Deploy to AWS EKS (kubectl) → Select env & image tag
```

**Prerequisites:**
- EKS cluster already created
- Kubernetes manifests in k8s/aws/manifests/
- GitHub Secrets: AWS_ROLE_ARN, RDS_MASTER_USER_SECRET_ARN, RDS_ENDPOINT, S3_VIDEOS_BUCKET, REDIS_ENDPOINT, OPENSEARCH_ENDPOINT, IRSA_METADATA_ROLE_ARN, IRSA_DATA_ROLE_ARN, CDN_DISTRIBUTION_ID, KAFKA_BROKERS

**Deployment time:** 2-5 minutes

---

## Comparison

| Feature | Terraform | kubectl |
|---------|-----------|---------|
| Infrastructure provisioning | ✅ Full | ❌ Requires existing |
| State management | ✅ Yes (S3) | ❌ No |
| Rollback capability | ✅ Yes | ⚠️ Manual |
| Speed | ⚠️ Slow (15-20 min) | ✅ Fast (2-5 min) |
| Complexity | ⚠️ High | ✅ Low |
| First-time setup cost | ⚠️ S3 bucket required | ✅ None |
| Team collaboration | ✅ Full audit trail | ⚠️ Basic |
| Version control | ✅ Infrastructure in git | ⚠️ Manifests only |

---

## Recommended Workflow

1. **First Deployment** → Use Terraform
   - Creates all infrastructure
   - Sets up VPC, EKS, databases, everything

2. **Daily Updates** → Use kubectl
   - Update image tags
   - Quick deployments
   - No infrastructure changes

3. **Infrastructure Changes** → Use Terraform
   - Add new services
   - Change scaling
   - Modify VPC settings

---

## Setup Guide

### Terraform Setup (One-time)

```bash
# 1. Create S3 bucket for state
aws s3api create-bucket \
  --bucket videostreamingplatform-terraform-state \
  --region us-east-1

# 2. Enable versioning
aws s3api put-bucket-versioning \
  --bucket videostreamingplatform-terraform-state \
  --versioning-configuration Status=Enabled

# 3. Create DynamoDB for locking
aws dynamodb create-table \
  --table-name terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

# 4. Add GitHub Secrets (Settings → Secrets):
#    - AWS_ROLE_ARN
#    - TERRAFORM_STATE_BUCKET
#    - TERRAFORM_LOCK_TABLE
#    - DATABASE_HOST
#    (RDS password is now AWS-managed via Secrets Manager — fetched at deploy time)
```

### kubectl Setup (Requires Existing Cluster)

```bash
# Assumes EKS cluster already exists
# Just add GitHub Secrets:
#   - AWS_ROLE_ARN
#   - RDS_MASTER_USER_SECRET_ARN  (from terraform output rds_master_user_secret_arn)
#   - RDS_ENDPOINT
```

---

## Files Organization

```
.github/workflows/
├── deploy-aws-terraform.yml     ← Full infrastructure
├── deploy-aws-k8s.yml           ← App-only updates
├── build.yml                    ← Build & push images
└── test.yml                     ← Run tests

terraform/aws/
├── provider.tf, variables.tf, outputs.tf
├── eks/main.tf                  ← EKS cluster
└── applications/main.tf         ← K8s resources

k8s/aws/manifests/
├── configmap.yaml, services.yaml
├── metadata-service-deploy.yaml
└── data-service-deploy.yaml
```

---

## Decision Tree

```
Q: Do you have an EKS cluster already?
├─ NO  → Use: Terraform (creates cluster)
└─ YES → Use: kubectl (deploys to cluster)

Q: Need to update only container images?
├─ YES → Use: kubectl (faster)
└─ NO  → Use: Terraform (infrastructure changes)

Q: Is this production first-time deploy?
├─ YES → Use: Terraform (full setup)
└─ NO  → Use: kubectl (app updates)
```

---

## Secrets Required

### Terraform Workflow
```
AWS_ROLE_ARN                  ✅ Required
TERRAFORM_STATE_BUCKET        ✅ Required
TERRAFORM_LOCK_TABLE          ✅ Required
DATABASE_HOST                 ✅ Required
SLACK_WEBHOOK                 ⚠️ Optional
(RDS password: AWS-managed in Secrets Manager — no GitHub secret needed)
```

### kubectl Workflow
```
AWS_ROLE_ARN                  ✅ Required
RDS_MASTER_USER_SECRET_ARN    ✅ Required (from `terraform output rds_master_user_secret_arn`)
RDS_ENDPOINT                  ✅ Required
S3_VIDEOS_BUCKET              ✅ Required
REDIS_ENDPOINT                ✅ Required
OPENSEARCH_ENDPOINT           ✅ Required
IRSA_METADATA_ROLE_ARN        ✅ Required
IRSA_DATA_ROLE_ARN            ✅ Required
CDN_DISTRIBUTION_ID           ✅ Required
KAFKA_BROKERS                 ✅ Required
SLACK_WEBHOOK                 ⚠️ Optional
```

---

## Troubleshooting

### Terraform Fails
```bash
cd terraform/aws
terraform plan -var-file="dev.tfvars" \
  -var="database_password=$DB_PW" \
  -var="database_host=$DB_HOST"
terraform destroy  # Clean up if needed
```

### kubectl Fails
```bash
aws eks update-kubeconfig --region us-east-1 --name videostreamingplatform-dev
kubectl get pods -n videostreamingplatform
kubectl describe pod <pod-name> -n videostreamingplatform
kubectl logs <pod-name> -n videostreamingplatform
```

---

## Documentation

- **Detailed Terraform guide** → `docs/TERRAFORM_AWS_DEPLOYMENT.md`
- **Deployment strategies** → `docs/DEPLOYMENT_STRATEGIES.md`
- **Secrets management** → `docs/AWS_DEPLOYMENT_SECRETS.md`

---

## Next Steps

1. Choose deployment strategy (Terraform or kubectl)
2. Set up secrets in GitHub
3. Test locally with terraform plan (if using Terraform)
4. Trigger workflow in GitHub Actions
5. Monitor deployment

Both workflows available - pick the right tool for each task.
