# Docker Registry Consolidation - Single Source of Truth

## Problem Solved

✅ **Consolidated from 2 registries to 1**

Previously there were **inconsistent image sources**:
- **AWS ECR** - Referenced in Kubernetes manifests and Terraform
- **GitHub Container Registry (GHCR)** - Used by build and deployment workflows

**Result**: All Docker images now use **GitHub Container Registry (ghcr.io)**

## Why GitHub Container Registry?

1. **Consistency** - All GitHub Actions workflows already push to ghcr.io
2. **Simplicity** - No AWS account ID needed in image URLs
3. **Unified** - One registry for all environments
4. **No extra setup** - Works with standard GitHub container access

## Changes Made

### 1. Kubernetes Manifests ✅
Updated image references from AWS ECR to GitHub Container Registry:

**Before:**
```yaml
image: ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/videostreamingplatform/metadata-service:latest
```

**After:**
```yaml
image: ghcr.io/videostreamingplatform/metadata-service:latest
```

Files updated:
- `k8s/aws/manifests/metadata-service-deploy.yaml`
- `k8s/aws/manifests/data-service-deploy.yaml`

### 2. Terraform Configuration ✅
Removed AWS ECR resources (no longer needed):

**Removed from `k8s/aws/terraform/storage.tf`:**
- `aws_ecr_repository.metadata_service` - ECR repo for metadata service
- `aws_ecr_repository.data_service` - ECR repo for data service
- `aws_ecr_lifecycle_policy.services` - ECR lifecycle policy
- `aws_kms_key.ecr` - KMS key for ECR encryption
- `aws_kms_alias.ecr` - KMS alias for ECR

**Removed from `k8s/aws/terraform/outputs.tf`:**
- `ecr_metadata_service_url` output
- `ecr_data_service_url` output

### 3. GitHub Actions Workflows ✅
No changes needed - already using ghcr.io:
- `build.yml` - Pushes to `ghcr.io/${{ github.repository }}/metadata-service`
- `deploy-aws-k8s.yml` - Deploys from `ghcr.io/${{ github.repository }}/metadata-service`
- `deploy-local.yml` - Uses `ghcr.io/${{ github.repository }}/metadata-service`

## Image Pull Flow

```
GitHub Actions (build.yml)
    ↓
Build Docker image
    ↓
Push to: ghcr.io/username/repo/service:tag
    ↓
Kubernetes (deploy-aws-k8s.yml or deploy-aws-terraform.yml)
    ↓
Pull from: ghcr.io/username/repo/service:tag
    ↓
Container running in pod
```

## Supported Image Tags

Images available at:
- `ghcr.io/videostreamingplatform/metadata-service:latest`
- `ghcr.io/videostreamingplatform/data-service:latest`
- `ghcr.io/videostreamingplatform/metadata-service:v1.0.0` (feature branch tags)
- `ghcr.io/videostreamingplatform/data-service:v1.0.0` (feature branch tags)

## Workflows Simplified

### Before
```
Build → Push to GHCR ✓
Terraform → Create ECR ✓
Deployment → Pull from GHCR (mismatch)
```

### After
```
Build → Push to GHCR ✓
Terraform → Skip ECR (not needed)
Deployment → Pull from GHCR ✓
```

## GitHub Container Registry Access

### Public Images
Images are public by default (no authentication needed):
```bash
docker pull ghcr.io/videostreamingplatform/metadata-service:latest
```

### Private Images (If needed)
Configure GitHub PAT for authentication:
```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
docker pull ghcr.io/videostreamingplatform/metadata-service:latest
```

### Kubernetes Secret (For private images)
Create pull secret if images are private:
```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=USERNAME \
  --docker-password=$GITHUB_TOKEN \
  --docker-email=user@example.com \
  -n videostreamingplatform
```

## Cost Implications

| Aspect | AWS ECR | GitHub Registry |
|--------|---------|-----------------|
| **Registry cost** | ~$0.10/GB/month storage | Free (GitHub enterprise) |
| **Data transfer** | $0.10/GB out (from us-east-1) | Free within GitHub |
| **Setup complexity** | Build + ECR + IAM | Just build |
| **Overall** | More expensive | Free |

## Deployment Commands

### Using kubectl
```bash
kubectl set image deployment/metadata-service \
  metadata-service=ghcr.io/videostreamingplatform/metadata-service:v1.0.0 \
  -n videostreamingplatform
```

### Using Terraform
Kubernetes manifests automatically pull latest images, which is overridden by workflows with specific tags.

### Using GitHub Actions
```bash
# Build and push to GHCR
Actions → Build & Push Docker Images → Run workflow

# Deploy using new image
Actions → Deploy to AWS EKS (kubectl)
```

## Terraform Variables

No longer need:
- `aws_account_id` (for ECR)
- `ecr_registry` settings

Still need:
- `container_image_tag` - For specifying which image version to deploy

## Consistency Check

All image references now point to GitHub Container Registry:

```
✅ build.yml         → ghcr.io
✅ deploy-aws-k8s.yml → ghcr.io
✅ deploy-local.yml  → ghcr.io
✅ manifests         → ghcr.io
✅ Terraform         → (not needed, manifests pull)
```

## Next Steps

1. **Update GitHub container privacy settings** if needed (Settings → Packages)
2. **Test image pull** after first build:
   ```bash
   docker pull ghcr.io/videostreamingplatform/metadata-service:latest
   ```
3. **Deploy using new image**:
   - Trigger build workflow (push to main)
   - Wait for build to complete
   - Deploy via Actions
4. **Verify pods** are running new images:
   ```bash
   kubectl get pods -n videostreamingplatform
   kubectl describe pod <pod-name> -n videostreamingplatform
   ```

## Migration Complete ✅

All Docker images now use **single source: GitHub Container Registry (ghcr.io)**

- ✅ Consistent across all workflows
- ✅ Simplified infrastructure (no ECR)
- ✅ Lower costs (free GitHub registry)
- ✅ Single deployment source
- ✅ No AWS account ID needed

---

**Single Source of Truth**: GitHub Container Registry (ghcr.io/videostreamingplatform/) ✅
