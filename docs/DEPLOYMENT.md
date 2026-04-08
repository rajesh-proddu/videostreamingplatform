# Deployment Guide

This document describes how to build, test, and deploy the Video Streaming Platform.

## Quick Start

### Local Development with Docker Compose

```bash
# Start all services locally
docker-compose up -d

# Verify services are healthy
docker-compose ps

# View logs
docker-compose logs -f metadata-service
```

### Local Kubernetes with Kind

```bash
# Create Kind cluster
terraform -C k8s/local/terraform init
terraform -C k8s/local/terraform apply

# Build and deploy
make docker-build
make deploy-local

# Verify deployment
kubectl get pods -n videostreamingplatform
kubectl port-forward -n videostreamingplatform svc/metadata-service 8080:8080
curl http://localhost:8080/health
```

### AWS EKS Deployment

#### Prerequisites
- AWS account with appropriate credentials
- Terraform installed
- kubectl configured

#### Step 1: Plan Infrastructure

```bash
# For dev environment
make terraform-plan ENVIRONMENT=dev

# For prod environment
make terraform-plan ENVIRONMENT=prod
```

#### Step 2: Deploy Infrastructure

```bash
# Apply Terraform changes
make terraform-apply ENVIRONMENT=dev

# This creates:
# - VPC with public/private subnets
# - EKS cluster
# - RDS MySQL cluster
# - S3 bucket for videos
# - ECR repositories
# - CloudWatch logging
```

#### Step 3: Deploy Applications

```bash
# Get EKS credentials
aws eks update-kubeconfig --region us-east-1 --name videostreamingplatform-dev

# Deploy services
make deploy-aws ENVIRONMENT=dev

# Verify deployment
kubectl get svc -n videostreamingplatform
kubectl logs -l app=metadata-service -n videostreamingplatform
```

## Build Pipeline

### GitHub Actions Workflows

1. **test.yml** - Runs on PR
   - Linting (golangci-lint, go vet)
   - Unit tests
   - Build artifacts
   - Code coverage

2. **build.yml** - Runs on merge to main
   - Builds Docker images
   - Pushes to GitHub Container Registry (ghcr.io)
   - Tags with commit SHA and `latest`

3. **deploy-local.yml** - Manual workflow
   - Sets up Kind cluster
   - Loads Docker images
   - Deploys to Kind
   - Runs smoke tests

4. **deploy-aws.yml** - Manual workflow
   - Assumes AWS IAM role
   - Updates kubeconfig
   - Deploys applications
   - Verifies rollout

### Local Build

```bash
# Run tests
make test

# Run linters
make lint

# Build binaries
make build

# Build Docker images
make docker-build

# Push to registry
REGISTRY=myregistry make docker-push
```

## Database Setup

### Local (Docker Compose)
- MySQL 8.0 runs automatically
- Initialized with schema from `scripts/init-db.sql`
- Credentials: videouser/videopass

### AWS (RDS)
- Aurora MySQL 8.0
- Created by Terraform
- Multi-AZ deployment
- Automated backups (7 days)
- CloudWatch logs enabled

#### Connect to RDS from EKS

```bash
# Get RDS endpoint from Terraform output
RDS_ENDPOINT=$(terraform output -C k8s/aws/terraform rds_cluster_endpoint)

# Port-forward through a pod
kubectl run -it --image=mysql:8.0 --rm --restart=Never -- mysql -h $RDS_ENDPOINT -u admin -p videoplatform
```

## Storage

### Local (MinIO)
- S3-compatible storage
- Endpoint: http://localhost:9000 (API), http://localhost:9001 (Console)
- Credentials: minioadmin/minioadmin

### AWS (S3)
- Bucket created by Terraform
- Encryption: KMS
- Versioning: Enabled
- Lifecycle: Auto-delete videos after 90 days (dev) / 365 days (prod)
- Access: Via IRSA (IAM Roles for Service Accounts)

## Monitoring & Logging

### Local
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686
- **Elasticsearch**: http://localhost:9200

### AWS
- **CloudWatch Logs**: /aws/eks/videostreamingplatform-{environment}
- **CloudWatch Metrics**: Custom metrics via OpenTelemetry
- **X-Ray**: Distributed tracing

## Troubleshooting

### Deployment Issues

```bash
# Check pod logs
kubectl logs -n videostreamingplatform <pod-name>

# Describe pod for events
kubectl describe pod -n videostreamingplatform <pod-name>

# Check resource usage
kubectl top pods -n videostreamingplatform

# Check events
kubectl get events -n videostreamingplatform
```

### Database Connectivity

```bash
# Test from pod
kubectl exec -it -n videostreamingplatform <pod-name> -- sh

# From shell in pod:
mysql -h $MYSQL_HOST -u $MYSQL_USER -p$MYSQL_PASSWORD videoplatform -e "SELECT 1"
```

### S3/MinIO Access

```bash
# Test MinIO locally
aws s3 ls s3://videostreamingplatform --endpoint-url=http://localhost:9000

# Test S3 from EKS pod
kubectl exec -it -n videostreamingplatform <pod-name> -- sh
aws s3 ls s3://videostreamingplatform-videos-dev-*
```

## Cleanup

### Local Resources
```bash
docker-compose down -v  # Remove volumes too
rm -rf k8s/local/terraform/.terraform*
```

### AWS Resources
```bash
# Destroy all infrastructure
make terraform-destroy ENVIRONMENT=dev

# This will delete:
# - EKS cluster
# - RDS database
# - S3 bucket (if empty)
# - VPC and subnets
```

## Performance Tuning

### Kubernetes Resources
- Adjust `resources.requests` and `resources.limits` in deployment manifests
- Use Horizontal Pod Autoscaling (HPA) for traffic spikes
- Configure cluster autoscaling for EKS nodes

### Database
- Monitor slow queries in CloudWatch
- Adjust RDS instance class based on load
- Enable read replicas for high-traffic scenarios

### Storage
- Monitor S3 access patterns
- Consider CloudFront for cross-region downloads
- Use S3 Transfer Acceleration for faster uploads

## Security Policies

### Network
- EKS nodes in private subnets
- NAT Gateway for internet access
- Security groups restrict traffic

### Data
- RDS and S3 encrypted with KMS
- IAM roles for pod access (IRSA)
- Read-only root filesystem in containers

### Secrets Management
- Store DB credentials in AWS Secrets Manager
- Pod IAM roles avoid credential exposure
- Rotate credentials regularly

## References

- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [EKS Documentation](https://docs.aws.amazon.com/eks/)
- [RDS Aurora MySQL](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Aurora.html)
- [Kubernetes Manifests](https://kubernetes.io/docs/concepts/)
