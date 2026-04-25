#!/usr/bin/env bash
# AWS Bootstrap Script — creates prerequisites for Terraform deployments
# Usage: ./bootstrap.sh [--destroy] [--region us-east-1] [--repo owner/repo]
set -euo pipefail

# ─── Defaults ────────────────────────────────────────────────────────────────
STATE_BUCKET="videostreamingplatform-terraform-state"
LOCK_TABLE="terraform-locks"
AWS_REGION="${AWS_REGION:-us-east-1}"
GITHUB_REPO="${GITHUB_REPO:-}"
DESTROY=false

# ─── Parse arguments ─────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --destroy)  DESTROY=true; shift ;;
    --region)   AWS_REGION="$2"; shift 2 ;;
    --repo)     GITHUB_REPO="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--destroy] [--region REGION] [--repo OWNER/REPO]"
      echo ""
      echo "  --destroy   Tear down bootstrap resources"
      echo "  --region    AWS region (default: us-east-1)"
      echo "  --repo      GitHub repo for OIDC (e.g. rajesh-proddu/videostreamingplatform)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ─── Helpers ─────────────────────────────────────────────────────────────────
info()  { echo -e "\033[1;34m[INFO]\033[0m  $*"; }
ok()    { echo -e "\033[1;32m[OK]\033[0m    $*"; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
err()   { echo -e "\033[1;31m[ERROR]\033[0m $*"; exit 1; }

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null) || err "AWS CLI not configured or no credentials found"
info "AWS Account: ${ACCOUNT_ID}  Region: ${AWS_REGION}"

# ─── Destroy mode ────────────────────────────────────────────────────────────
if [ "$DESTROY" = true ]; then
  warn "Destroying bootstrap resources..."

  # Delete OIDC provider
  OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
  aws iam delete-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_ARN" 2>/dev/null && ok "Deleted OIDC provider" || warn "OIDC provider not found"

  # Detach policies and delete GitHub Actions role
  for ROLE_NAME in github-actions-deploy metadata-service-irsa data-service-irsa; do
    POLICIES=$(aws iam list-attached-role-policies --role-name "$ROLE_NAME" --query 'AttachedPolicies[].PolicyArn' --output text 2>/dev/null || true)
    for ARN in $POLICIES; do
      aws iam detach-role-policy --role-name "$ROLE_NAME" --policy-arn "$ARN" 2>/dev/null || true
    done
    INLINE=$(aws iam list-role-policies --role-name "$ROLE_NAME" --query 'PolicyNames[]' --output text 2>/dev/null || true)
    for PNAME in $INLINE; do
      aws iam delete-role-policy --role-name "$ROLE_NAME" --policy-name "$PNAME" 2>/dev/null || true
    done
    aws iam delete-role --role-name "$ROLE_NAME" 2>/dev/null && ok "Deleted role: $ROLE_NAME" || warn "Role $ROLE_NAME not found"
  done

  # Delete DynamoDB table
  aws dynamodb delete-table --table-name "$LOCK_TABLE" --region "$AWS_REGION" 2>/dev/null && ok "Deleted DynamoDB table: $LOCK_TABLE" || warn "Table $LOCK_TABLE not found"

  # Empty and delete S3 bucket
  if aws s3api head-bucket --bucket "$STATE_BUCKET" --region "$AWS_REGION" 2>/dev/null; then
    aws s3 rm "s3://${STATE_BUCKET}" --recursive --region "$AWS_REGION" 2>/dev/null || true
    # Delete versioned objects
    aws s3api list-object-versions --bucket "$STATE_BUCKET" --region "$AWS_REGION" \
      --query 'Versions[].{Key:Key,VersionId:VersionId}' --output json 2>/dev/null | \
      python3 -c "
import sys, json
objs = json.load(sys.stdin)
if objs:
    for o in objs:
        print(f\"{o['Key']} {o['VersionId']}\")
" 2>/dev/null | while read -r KEY VID; do
      aws s3api delete-object --bucket "$STATE_BUCKET" --key "$KEY" --version-id "$VID" --region "$AWS_REGION" 2>/dev/null || true
    done
    aws s3api delete-bucket --bucket "$STATE_BUCKET" --region "$AWS_REGION" 2>/dev/null && ok "Deleted S3 bucket: $STATE_BUCKET" || warn "Could not delete bucket"
  else
    warn "Bucket $STATE_BUCKET not found"
  fi

  ok "Bootstrap resources destroyed"
  exit 0
fi

# ═══════════════════════════════════════════════════════════════════════════════
# STEP 1: S3 Bucket for Terraform State
# ═══════════════════════════════════════════════════════════════════════════════
info "Step 1: Creating S3 bucket for Terraform state..."

if aws s3api head-bucket --bucket "$STATE_BUCKET" --region "$AWS_REGION" 2>/dev/null; then
  ok "Bucket already exists: $STATE_BUCKET"
else
  if [ "$AWS_REGION" = "us-east-1" ]; then
    aws s3api create-bucket --bucket "$STATE_BUCKET" --region "$AWS_REGION"
  else
    aws s3api create-bucket --bucket "$STATE_BUCKET" --region "$AWS_REGION" \
      --create-bucket-configuration LocationConstraint="$AWS_REGION"
  fi

  aws s3api put-bucket-versioning --bucket "$STATE_BUCKET" --region "$AWS_REGION" \
    --versioning-configuration Status=Enabled

  aws s3api put-bucket-encryption --bucket "$STATE_BUCKET" --region "$AWS_REGION" \
    --server-side-encryption-configuration '{
      "Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}, "BucketKeyEnabled": true}]
    }'

  aws s3api put-public-access-block --bucket "$STATE_BUCKET" --region "$AWS_REGION" \
    --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

  ok "Created S3 bucket: $STATE_BUCKET (versioned, encrypted, private)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# STEP 2: DynamoDB Table for State Locking
# ═══════════════════════════════════════════════════════════════════════════════
info "Step 2: Creating DynamoDB lock table..."

if aws dynamodb describe-table --table-name "$LOCK_TABLE" --region "$AWS_REGION" &>/dev/null; then
  ok "Table already exists: $LOCK_TABLE"
else
  aws dynamodb create-table \
    --table-name "$LOCK_TABLE" \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region "$AWS_REGION"

  aws dynamodb wait table-exists --table-name "$LOCK_TABLE" --region "$AWS_REGION"
  ok "Created DynamoDB table: $LOCK_TABLE"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# STEP 3: GitHub OIDC Identity Provider
# ═══════════════════════════════════════════════════════════════════════════════
info "Step 3: Creating GitHub OIDC identity provider..."

OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_ARN" &>/dev/null; then
  ok "OIDC provider already exists"
else
  THUMBPRINT=$(openssl s_client -servername token.actions.githubusercontent.com \
    -showcerts -connect token.actions.githubusercontent.com:443 </dev/null 2>/dev/null | \
    openssl x509 -fingerprint -sha1 -noout 2>/dev/null | \
    sed 's/://g' | cut -d= -f2 | tr '[:upper:]' '[:lower:]')

  aws iam create-open-id-connect-provider \
    --url "https://token.actions.githubusercontent.com" \
    --client-id-list "sts.amazonaws.com" \
    --thumbprint-list "$THUMBPRINT"

  ok "Created OIDC provider for GitHub Actions"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# STEP 4: GitHub Actions IAM Role
# ═══════════════════════════════════════════════════════════════════════════════
info "Step 4: Creating GitHub Actions IAM role..."

if [ -z "$GITHUB_REPO" ]; then
  # Try to infer from git remote
  GITHUB_REPO=$(git remote get-url origin 2>/dev/null | sed -E 's|.*github\.com[:/](.*)\.git$|\1|' | sed 's|\.git$||') || true
  if [ -z "$GITHUB_REPO" ]; then
    warn "Could not detect GitHub repo. Pass --repo owner/repo"
    warn "Skipping IAM role creation"
    SKIP_ROLES=true
  else
    info "Detected repo: $GITHUB_REPO"
    SKIP_ROLES=false
  fi
else
  SKIP_ROLES=false
fi

if [ "$SKIP_ROLES" = false ]; then
  DEPLOY_ROLE_NAME="github-actions-deploy"

  TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:${GITHUB_REPO}:*"
        }
      }
    }
  ]
}
EOF
)

  if aws iam get-role --role-name "$DEPLOY_ROLE_NAME" &>/dev/null; then
    ok "Role already exists: $DEPLOY_ROLE_NAME"
    aws iam update-assume-role-policy --role-name "$DEPLOY_ROLE_NAME" --policy-document "$TRUST_POLICY"
    ok "Updated trust policy"
  else
    aws iam create-role \
      --role-name "$DEPLOY_ROLE_NAME" \
      --assume-role-policy-document "$TRUST_POLICY" \
      --description "GitHub Actions deployment role for ${GITHUB_REPO}"
    ok "Created role: $DEPLOY_ROLE_NAME"
  fi

  # Attach policies for EKS, ECR, S3, RDS, VPC, DynamoDB, CloudWatch, IAM (for IRSA)
  DEPLOY_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EKSAccess",
      "Effect": "Allow",
      "Action": [
        "eks:DescribeCluster",
        "eks:ListClusters",
        "eks:UpdateClusterConfig",
        "eks:UpdateClusterVersion",
        "eks:DescribeUpdate",
        "eks:ListUpdates",
        "eks:CreateCluster",
        "eks:DeleteCluster",
        "eks:CreateNodegroup",
        "eks:DeleteNodegroup",
        "eks:DescribeNodegroup",
        "eks:ListNodegroups",
        "eks:UpdateNodegroupConfig",
        "eks:UpdateNodegroupVersion",
        "eks:TagResource",
        "eks:UntagResource",
        "eks:AssociateEncryptionConfig",
        "eks:ListTagsForResource"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2VPCAccess",
      "Effect": "Allow",
      "Action": [
        "ec2:*Vpc*", "ec2:*Subnet*", "ec2:*SecurityGroup*",
        "ec2:*InternetGateway*", "ec2:*NatGateway*", "ec2:*RouteTable*",
        "ec2:*Address*", "ec2:*Tags*", "ec2:Describe*",
        "ec2:AllocateAddress", "ec2:ReleaseAddress",
        "ec2:CreateRoute", "ec2:DeleteRoute"
      ],
      "Resource": "*"
    },
    {
      "Sid": "RDSAccess",
      "Effect": "Allow",
      "Action": [
        "rds:*DBCluster*", "rds:*DBInstance*", "rds:*DBSubnetGroup*",
        "rds:AddTagsToResource", "rds:ListTagsForResource",
        "rds:DescribeDBEngineVersions", "rds:DescribeOrderableDBInstanceOptions"
      ],
      "Resource": "*"
    },
    {
      "Sid": "S3Access",
      "Effect": "Allow",
      "Action": ["s3:*"],
      "Resource": [
        "arn:aws:s3:::videostreamingplatform-*",
        "arn:aws:s3:::videostreamingplatform-*/*"
      ]
    },
    {
      "Sid": "DynamoDBLock",
      "Effect": "Allow",
      "Action": ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem"],
      "Resource": "arn:aws:dynamodb:*:${ACCOUNT_ID}:table/terraform-locks"
    },
    {
      "Sid": "IAMRoles",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole", "iam:CreateRole", "iam:DeleteRole", "iam:PassRole",
        "iam:AttachRolePolicy", "iam:DetachRolePolicy",
        "iam:PutRolePolicy", "iam:DeleteRolePolicy",
        "iam:GetRolePolicy", "iam:ListAttachedRolePolicies",
        "iam:ListRolePolicies", "iam:TagRole", "iam:UntagRole",
        "iam:GetOpenIDConnectProvider", "iam:CreateOpenIDConnectProvider",
        "iam:DeleteOpenIDConnectProvider", "iam:ListOpenIDConnectProviders",
        "iam:TagOpenIDConnectProvider", "iam:UntagOpenIDConnectProvider",
        "iam:CreatePolicy", "iam:DeletePolicy", "iam:GetPolicy",
        "iam:ListPolicyVersions", "iam:CreatePolicyVersion", "iam:DeletePolicyVersion",
        "iam:GetPolicyVersion",
        "iam:CreateServiceLinkedRole"
      ],
      "Resource": "*"
    },
    {
      "Sid": "KMS",
      "Effect": "Allow",
      "Action": [
        "kms:CreateKey", "kms:DescribeKey", "kms:EnableKeyRotation",
        "kms:GetKeyPolicy", "kms:GetKeyRotationStatus",
        "kms:ListResourceTags", "kms:ScheduleKeyDeletion",
        "kms:TagResource", "kms:CreateAlias", "kms:DeleteAlias",
        "kms:ListAliases", "kms:CreateGrant", "kms:Encrypt",
        "kms:Decrypt", "kms:GenerateDataKey", "kms:GenerateDataKey*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CloudWatchLogs",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup", "logs:DeleteLogGroup",
        "logs:DescribeLogGroups", "logs:PutRetentionPolicy",
        "logs:TagLogGroup", "logs:ListTagsLogGroup",
        "logs:TagResource", "logs:ListTagsForResource"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SecretsManagerRDSManagedPassword",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:rds!db-*"
    }
  ]
}
EOF
)

  aws iam put-role-policy \
    --role-name "$DEPLOY_ROLE_NAME" \
    --policy-name "deploy-policy" \
    --policy-document "$DEPLOY_POLICY"
  ok "Attached deployment policy"

  DEPLOY_ROLE_ARN=$(aws iam get-role --role-name "$DEPLOY_ROLE_NAME" --query 'Role.Arn' --output text)
fi

# ═══════════════════════════════════════════════════════════════════════════════
# STEP 5: IRSA Roles (created by Terraform, but we output what's needed)
# ═══════════════════════════════════════════════════════════════════════════════
info "Step 5: IRSA roles will be created by Terraform (eks.tf)"
info "  - metadata-service-irsa: RDS read/write access"
info "  - data-service-irsa: S3 read/write access"
ok "IRSA roles are managed by Terraform — no bootstrap action needed"

# ═══════════════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  ✅  AWS Bootstrap Complete"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  S3 State Bucket:    ${STATE_BUCKET}"
echo "  DynamoDB Lock Table: ${LOCK_TABLE}"
echo "  OIDC Provider:       token.actions.githubusercontent.com"
if [ "${SKIP_ROLES:-false}" = false ]; then
echo "  Deploy Role ARN:     ${DEPLOY_ROLE_ARN}"
fi
echo ""
echo "─── Required GitHub Secrets ─────────────────────────────────"
echo ""
echo "  Set these in: https://github.com/${GITHUB_REPO}/settings/secrets/actions"
echo ""
echo "  AWS_ROLE_ARN              = ${DEPLOY_ROLE_ARN:-<create role first>}"
echo "  TERRAFORM_STATE_BUCKET    = ${STATE_BUCKET}"
echo "  TERRAFORM_LOCK_TABLE      = ${LOCK_TABLE}"
echo ""
echo "  After first terraform apply, also set (for deploy-aws-k8s.yml):"
echo "  RDS_MASTER_USER_SECRET_ARN = <from: terraform output rds_master_user_secret_arn>"
echo "  (RDS master password is now AWS-managed via Secrets Manager — no separate secret needed)"
echo ""
echo "─── Optional Secrets ────────────────────────────────────────"
echo ""
echo "  SLACK_WEBHOOK              = <Slack incoming webhook URL>"
echo ""
echo "─── Next Steps ──────────────────────────────────────────────"
echo ""
echo "  1. Set the GitHub secrets listed above"
echo "  2. Run: gh workflow run deploy-aws-terraform.yml -f environment=dev -f image_tag=latest"
echo "  3. Terraform will create EKS, RDS, S3, VPC, and IRSA roles"
echo "  4. Then run: gh workflow run deploy-aws-k8s.yml -f environment=dev -f image_tag=latest"
echo ""
