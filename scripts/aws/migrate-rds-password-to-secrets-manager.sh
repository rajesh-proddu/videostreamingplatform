#!/usr/bin/env bash
# Migrates RDS master password from GitHub secret (DATABASE_PASSWORD) to
# AWS-managed Secrets Manager, and updates GitHub repo secrets accordingly.
#
# Prereqs:
#   - terraform & gh CLI installed and authenticated
#   - AWS credentials active (aws sts get-caller-identity must work)
#   - Run from repo root
#
# Usage:  scripts/aws/migrate-rds-password-to-secrets-manager.sh dev|prod

set -euo pipefail

ENV="${1:-}"
if [[ "$ENV" != "dev" && "$ENV" != "prod" ]]; then
  echo "Usage: $0 dev|prod" >&2
  exit 1
fi

REPO="rajesh-proddu/videostreamingplatform"
TF_DIR="k8s/aws/terraform"

echo "── Preflight ────────────────────────────────────────────────"
aws sts get-caller-identity >/dev/null || { echo "AWS creds not active" >&2; exit 1; }
gh auth status >/dev/null 2>&1          || { echo "gh not authenticated" >&2; exit 1; }
terraform -chdir="$TF_DIR" version >/dev/null
echo "  ✓ AWS, gh, terraform all ready"
echo ""

echo "── Step 1: terraform plan (review before applying) ──────────"
terraform -chdir="$TF_DIR" init -upgrade
terraform -chdir="$TF_DIR" plan \
  -var-file="terraform.${ENV}.tfvars" \
  -out="tfplan.${ENV}"
echo ""
read -rp "Review the plan above. Apply? [y/N] " CONFIRM
[[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]] || { echo "Aborted."; exit 0; }

echo "── Step 2: terraform apply ──────────────────────────────────"
terraform -chdir="$TF_DIR" apply "tfplan.${ENV}"

echo "── Step 3: capture ARN and set GitHub secret ────────────────"
SECRET_ARN=$(terraform -chdir="$TF_DIR" output -raw rds_master_user_secret_arn)
echo "  ARN: $SECRET_ARN"
gh secret set RDS_MASTER_USER_SECRET_ARN --repo "$REPO" --body "$SECRET_ARN"
echo "  ✓ GitHub secret RDS_MASTER_USER_SECRET_ARN set"
echo ""

echo "── Step 4: sanity check — fetch secret via AWS CLI ──────────"
aws secretsmanager get-secret-value --secret-id "$SECRET_ARN" \
  --query 'SecretString' --output text | jq -r '.username' >/dev/null
echo "  ✓ Secret is readable (username field present)"
echo ""

echo "── Step 5: verify new deploy flow before deleting old secret ─"
echo "  Trigger the 'Deploy to AWS EKS (Terraform)' workflow in GitHub Actions"
echo "  and confirm it completes successfully. Then run:"
echo ""
echo "    gh secret delete DATABASE_PASSWORD --repo $REPO"
echo ""
echo "  (Not done automatically — destructive and unrecoverable.)"
