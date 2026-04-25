# AWS Deployment Strategies - Comparison & Guide

Your repository now has **TWO deployment workflows** - choose the one that fits your needs.

## Workflow Files

| File | Method | Use Case |
|------|--------|----------|
| `deploy-aws-terraform.yml` | Terraform IaC | Production, version control, rollback capabilities |
| `deploy-aws-k8s.yml` | kubectl direct | Quick deployments, existing manifests, simple updates |

## Side-by-Side Comparison

### Setup & Prerequisites

| Aspect | Terraform | kubectl |
|--------|-----------|---------|
| **Learning curve** | Medium (HCL language) | Low (Kubernetes YAML) |
| **Setup time** | Higher (S3 backend, state management) | Lower (just kubectl) |
| **Infrastructure** | Full stack (VPC, EKS, nodes, apps) | Assumes cluster exists |
| **Tools needed** | terraform CLI | kubectl CLI |

### How They Work

**Terraform Workflow:**
```
1. Checkout code
2. Configure AWS credentials (IAM role)
3. Initialize Terraform with S3 backend
4. Plan changes (shows what will be created/modified)
5. Apply changes (creates/updates everything)
6. Test deployment
```

**kubectl Workflow:**
```
1. Checkout code
2. Configure AWS credentials (IAM role)
3. Configure kubectl with EKS cluster
4. Apply Kubernetes manifests (deployments, services, etc.)
5. Update container images
6. Test deployment
```

### State Management

| Aspect | Terraform | kubectl |
|--------|-----------|---------|
| **State tracking** | ✅ S3 backend with locking | ❌ None (cluster is source of truth) |
| **Version history** | ✅ Full history (git + tfstate) | ⚠️ Git history only |
| **Rollback** | ✅ `terraform destroy` + redeploy | ⚠️ Manual or git revert |
| **Concurrent deployments** | ✅ Locked to prevent conflicts | ❌ No protection |

### Infrastructure Management

| Feature | Terraform | kubectl |
|---------|-----------|---------|
| **VPC creation** | ✅ Full VPC setup | ❌ Requires external VPC |
| **EKS cluster** | ✅ Creates & manages | ❌ Requires existing cluster |
| **Node groups** | ✅ Auto-scaling configured | ❌ Requires existing nodes |
| **Networking** | ✅ Subnets, NAT, routing | ❌ Requires existing setup |
| **Deployments** | ✅ via Kubernetes provider | ✅ Direct manifests |
| **Services** | ✅ Declarative | ✅ Declarative |
| **ConfigMaps** | ✅ Declarative | ✅ Declarative |
| **Secrets** | ✅ Declarative | ✅ Declarative |

## When to Use Each

### Use Terraform If:

✅ **Starting from scratch** - Need to create VPC, EKS, nodes, everything  
✅ **Multiple environments** - dev/prod with different configurations  
✅ **Version control infrastructure** - Track all changes in git  
✅ **Team development** - Need state locking and collaboration  
✅ **Production grade** - Need audit trail and rollback capability  
✅ **Complex deployments** - Multiple AWS resources (RDS, S3, IAM, etc.)  

### Use kubectl If:

✅ **Cluster already exists** - Just deploying to existing infrastructure  
✅ **Quick deployments** - Simple application updates  
✅ **Existing manifests** - Already have YAML files in k8s/aws/manifests/  
✅ **Learning/testing** - Don't need full infrastructure changes  
✅ **Container updates only** - Just changing image tags  
✅ **Simple workflows** - No need for state management  

## Hybrid Approach (Recommended for Most Teams)

```
Phase 1: Infrastructure Setup
├─ Use: Terraform (one time)
└─ Creates: VPC, EKS, node groups, databases, storage

Phase 2: Application Updates
├─ Use: kubectl (frequent)
└─ Updates: Image tags, replicas, configurations
```

## File Organization

```
.github/workflows/
├── deploy-aws-terraform.yml      # Full infrastructure deployment
├── deploy-aws-k8s.yml            # Application-only deployment
├── build.yml                      # Build and push images
└── test.yml                       # Run tests

terraform/aws/
├── provider.tf                    # Provider configuration
├── variables.tf                   # Variables
├── outputs.tf                     # Outputs
├── eks/main.tf                    # EKS infrastructure
├── applications/main.tf           # Kubernetes deployments
├── dev.tfvars                     # Dev environment
└── prod.tfvars                    # Prod environment

k8s/aws/manifests/
├── configmap.yaml                # Application config
├── services.yaml                  # Kubernetes services
├── metadata-service-deploy.yaml   # Metadata service
└── data-service-deploy.yaml       # Data service
```

## Usage Examples

### Scenario 1: Initial Production Deployment

```bash
# Use Terraform to create everything
GitHub Actions:
1. Click "Actions" → "Deploy to AWS EKS (Terraform)"
2. Environment: prod
3. Image tag: v1.0.0
4. Run workflow

# This creates:
- VPC with subnets
- EKS cluster
- Node groups
- RDS database
- Kubernetes resources (deployments, services, etc.)
```

### Scenario 2: Quick Application Update

```bash
# Use kubectl to update image

GitHub Actions:
1. Click "Actions" → "Deploy to AWS EKS (kubectl)"
2. Environment: prod
3. Image tag: v1.0.1
4. Run workflow

# This only:
- Updates container image
- Triggers rolling deployment
# Much faster! (1-2 minutes vs 15-20 minutes)
```

### Scenario 3: Scale Services

**With Terraform:**
```hcl
# Edit prod.tfvars
data_service_replicas = 5  # was 3
metadata_service_replicas = 5

# Git commit and trigger workflow
terraform destroy + terraform apply
# Full redeployment
```

**With kubectl:**
```bash
# In workflow, add step:
kubectl scale deployment data-service --replicas=5 -n videostreamingplatform
kubectl scale deployment metadata-service --replicas=5 -n videostreamingplatform
# Instant scaling
```

### Scenario 4: Rollback After Bad Deploy

**With Terraform:**
```bash
# Option 1: Use previous terraform state
terraform state pull > previous.tfstate
# Restore and apply

# Option 2: Git revert
git revert <commit>
git push
# Trigger workflow - applies previous version
```

**With kubectl:**
```bash
# Kubernetes built-in rollback
kubectl rollout history deployment/data-service -n videostreamingplatform
kubectl rollout undo deployment/data-service -n videostreamingplatform
# Instant rollback to previous image
```

## GitHub Actions Decision Tree

```
┌─ Start: Need to deploy?
│
├─ Infrastructure changes (VPC, EKS, nodes)?
│  └─ YES → Use deploy-aws-terraform.yml
│
├─ Application/image changes only?
│  └─ YES → Use deploy-aws-k8s.yml (faster)
│
├─ Scaling services?
│  ├─ Terraform? → deploy-aws-terraform.yml (full redeployment)
│  └─ kubectl? → deploy-aws-k8s.yml + kubectl scale
│
└─ First time production?
   └─ Use deploy-aws-terraform.yml (creates everything)
```

## Required Secrets Setup

### For Terraform Workflow

```
AWS_ROLE_ARN                    # IAM role for GitHub
TERRAFORM_STATE_BUCKET          # S3 bucket for state
TERRAFORM_LOCK_TABLE            # DynamoDB for locking
DATABASE_HOST                   # RDS endpoint
SLACK_WEBHOOK                   # Optional notifications
# RDS password: AWS-managed in Secrets Manager — no GitHub secret needed
```

### For kubectl Workflow

```
AWS_ROLE_ARN                    # IAM role for GitHub
RDS_MASTER_USER_SECRET_ARN      # Secrets Manager ARN (from terraform output)
SLACK_WEBHOOK                   # Optional notifications
```

kubectl requires **fewer secrets** because infrastructure already exists.

## Security Considerations

### Terraform
- ✅ State files encrypted in S3
- ✅ DynamoDB locking prevents conflicts
- ✅ IAM role-based access
- ✅ Full audit trail (git + tfstate)
- ⚠️ Requires S3 bucket setup
- ⚠️ More infrastructure = larger attack surface

### kubectl
- ✅ Simpler (fewer AWS resources)
- ✅ Cluster-level RBAC only
- ✅ Faster security reviews
- ⚠️ No state backup unless in git
- ⚠️ Manual rollback needed

## Cost Implications

### Terraform Setup
- **One-time**: S3 bucket (~$0.02/month), DynamoDB table (~$0.25/month)
- **Each deployment**: Standard AWS charges (EKS, EC2, RDS, etc.)

### kubectl Setup
- **Assumes**: Existing infrastructure paid for
- **Only updates**: Application deployment (minimal additional cost)

## Troubleshooting

### Terraform Issues

```bash
# State lock stuck?
terraform force-unlock LOCK_ID

# State mismatch?
terraform refresh && terraform plan

# Want to see what exists?
terraform state list
terraform state show aws_eks_cluster.videostreamingplatform
```

### kubectl Issues

```bash
# Deployment stuck?
kubectl describe deployment data-service -n videostreamingplatform
kubectl logs -l app=data-service -n videostreamingplatform

# Want to rollback?
kubectl rollout undo deployment/data-service -n videostreamingplatform
```

## Migration Path

```
Current State (kubectl):
  ↓
  └─ Start using Terraform for infrastructure
  └─ Keep kubectl for app updates
  └─ Both workflows coexist
  ↓
Learn from use:
  ├─ If Terraform works well → primary method
  ├─ If kubectl is enough → keep it simple
  └─ Eventually standardize on one
```

## Recommendations

### For New Projects
→ **Use Terraform** (build good habits from start)

### For Existing Clusters
→ **Start with kubectl** (quick wins), **Migrate to Terraform** (long-term)

### For Teams
→ **Use both** (dev updates with kubectl, releases with Terraform)

### For Production
→ **Terraform for infrastructure**, **kubectl for app updates**

## Documentation References

- **Terraform deployment**: `docs/TERRAFORM_AWS_DEPLOYMENT.md`
- **AWS Secrets strategy**: `docs/AWS_DEPLOYMENT_SECRETS.md`
- **Kubernetes patterns**: `k8s/aws/manifests/` (manifest files)

## Quick Decision Matrix

| Situation | Use |
|-----------|-----|
| "First time deploying to AWS" | Terraform |
| "Just update app image" | kubectl |
| "Change replicas" | Either (Terraform is tracked, kubectl is instant) |
| "Add new AWS resource (RDS, etc)" | Terraform |
| "Emergency hotfix" | kubectl |
| "Team reviewing changes" | Terraform (plan visible in PR) |
| "Testing locally" | Terraform (reproducible) |
| "Really in hurry" | kubectl (1-2 min vs 15-20) |

---

**Summary**: You now have flexibility to choose the right tool for each situation. Start with Terraform for one-time infrastructure setup, then use kubectl for day-to-day application updates.
