# AWS Deployment - Secret Management Strategy

## ✅ Your Deployment is Secure

Your `.github/workflows/deploy-aws.yml` uses **AWS IAM Role assumption** - the most secure method for GitHub Actions + AWS integration.

## How It Works

### 1. **AWS IAM Role Assumption** (No Hardcoded Keys)
```yaml
permissions:
  id-token: write        # GitHub provides OIDC token
  contents: read

steps:
  - name: Configure AWS credentials
    uses: aws-actions/configure-aws-credentials@v4
    with:
      role-to-assume: ${{ secrets.AWS_ROLE_ARN }}  # Only ARN stored in GitHub
      aws-region: ${{ env.AWS_REGION }}
```

**What happens:**
1. GitHub generates an OIDC token proving the workflow is legitimate
2. Token is exchanged for temporary AWS credentials (15-60 min lifetime)
3. No long-lived access keys in repository
4. Automatic credential rotation

### 2. **Environment-Based Secrets**
```yaml
environment: 
  name: ${{ github.event.inputs.environment }}  # dev or prod
```

This allows different secrets/permissions per environment:
- `dev` environment → dev AWS credentials
- `prod` environment → prod AWS credentials (more restrictive)

### 3. **RDS/Database Secrets**
```yaml
- name: Deploy configmaps and secrets
  run: |
    kubectl apply -f k8s/aws/manifests/configmap.yaml
    # Secrets should be managed separately via AWS Secrets Manager or sealed-secrets
```

**Current approach is good but can be improved:**

**Option A: AWS Secrets Manager**
```bash
# Store secrets in AWS
aws secretsmanager create-secret \
  --name videostreamingplatform/prod/db-password \
  --secret-string "your-password"

# Kubernetes pulls from AWS
kubectl create secret generic db-secrets \
  --from-literal=password=$(aws secretsmanager get-secret-value --secret-id videostreamingplatform/prod/db-password --query SecretString --output text)
```

**Option B: Sealed Secrets (Recommended for K8s)**
```bash
# Install sealed-secrets controller in cluster
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml

# Create and seal secret
echo -n "your-password" | kubectl create secret generic db-secrets --dry-run=client --from-file=password=/dev/stdin -o yaml | kubeseal -f - > sealed-db-secrets.yaml

# Commit sealed version (can't be decrypted outside cluster)
git add k8s/aws/manifests/sealed-db-secrets.yaml
kubectl apply -f k8s/aws/manifests/sealed-db-secrets.yaml
```

## Security Layers

```
┌─────────────────────────────────────┐
│  GitHub Actions Workflow            │
│  (Restricted by branch/environment) │
└────────────┬────────────────────────┘
             │
             ├─→ OIDC Token (temporary)
             │
┌────────────▼──────────────────────────┐
│  AWS IAM (assume role)                │
│  - Temporary credentials (15-60 min)  │
│  - Environment-specific permissions   │
│  - Can be denied by MFA or IP         │
└────────────┬───────────────────────────┘
             │
             ├─→ AWS Secrets Manager (optional)
             │   └─→ Encrypted database passwords
             │
             ├─→ EKS Cluster
             │   └─→ Sealed Secrets or AWS Secrets integration
             │
             └─→ Application pods receive secrets
```

## Your Setup Validation

| Component | Current | Recommendation | Notes |
|-----------|---------|-----------------|-------|
| AWS Access | ✅ OIDC Role | ✅ Best practice | No hardcoded keys |
| Permissions | ✅ Minimal (id-token, contents) | ✅ Correct | Only what's needed |
| Environments | ✅ dev/prod split | ✅ Correct | Different permissions per env |
| K8s Secrets | ⚠️ ConfigMap/manual | 📈 Upgrade to Sealed Secrets | Encrypt secrets at rest |
| Database Secrets | ⚠️ Not shown | 📈 Use AWS Secrets Manager | Centralized secret rotation |

## Recommended Improvements

### 1. **Add AWS Secrets Manager Integration**
```yaml
- name: Fetch secrets from AWS Secrets Manager
  run: |
    DB_PASSWORD=$(aws secretsmanager get-secret-value \
      --secret-id videostreamingplatform/${{ github.event.inputs.environment }}/db-password \
      --query SecretString --output text)
    
    kubectl create secret generic db-secrets \
      --from-literal=password=$DB_PASSWORD \
      -n videostreamingplatform --dry-run=client -o yaml | kubectl apply -f -
```

### 2. **Add Sealed Secrets for K8s**
```yaml
- name: Deploy sealed secrets
  run: |
    kubectl apply -f k8s/aws/manifests/sealed-db-secrets.yaml
    kubectl apply -f k8s/aws/manifests/sealed-cache-secrets.yaml
```

### 3. **Store Secrets in AWS Secrets Manager**
```bash
# Create dev database secret
aws secretsmanager create-secret \
  --name videostreamingplatform/dev/db-secrets \
  --secret-string '{
    "username": "videouser",
    "password": "dev-password",
    "host": "rds-dev.c...rds.amazonaws.com"
  }'

# Create prod database secret (different password!)
aws secretsmanager create-secret \
  --name videostreamingplatform/prod/db-secrets \
  --secret-string '{
    "username": "videouser",
    "password": "prod-password",
    "host": "rds-prod.c...rds.amazonaws.com"
  }'
```

### 4. **Add Environment Protection Rules**
In GitHub Settings → Environments → prod:
- ✅ Require reviewers for approval
- ✅ Restrict to main branch
- ✅ Add deployment reviewers (team leads)

## Secret Lifecycle

```
1. Developer creates secret in AWS Secrets Manager
   aws secretsmanager create-secret --name videostreamingplatform/prod/api-key

2. GitHub determines environment (dev or prod) from workflow input
   environment: 
     name: ${{ github.event.inputs.environment }}

3. Workflow assumes IAM role with permission to read secrets
   role-to-assume: ${{ secrets.AWS_ROLE_ARN }}

4. Workflow fetches secret securely from AWS
   aws secretsmanager get-secret-value --secret-id videostreamingplatform/prod/api-key

5. Secret is passed to Kubernetes securely
   kubectl create secret generic app-secrets --from-literal=key=$SECRET

6. Pod mounts secret as volume (never in environment variables)
   volumeMounts:
     - name: app-secrets
       mountPath: /etc/secrets/
       readOnly: true

7. Application reads from file, not environment
   password := os.ReadFile("/etc/secrets/password")
```

## Next Steps

1. **Store database passwords in AWS Secrets Manager** (not in git)
2. **Use Sealed Secrets for K8s** (encrypt secrets at rest in etcd)
3. **Add environment protection rules** (require approvals for prod)
4. **Remove any credentials from config files** (use AWS Secrets Manager instead)
5. **Audit secret access** (CloudTrail logs all secret retrievals)

## References

- [AWS IAM OIDC with GitHub](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [AWS Secrets Manager](https://docs.aws.amazon.com/secretsmanager/)
- [Bitnami Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- [Kubernetes Secrets Best Practices](https://kubernetes.io/docs/concepts/configuration/secret/)
