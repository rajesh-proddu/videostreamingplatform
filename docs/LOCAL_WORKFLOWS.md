# Running GitHub Workflows Locally with `act`

## Quick Start

### Installation

The `act` tool has been installed in `./bin/act`. To use globally, add to PATH or use with `./bin/act`.

```bash
# Make accessible globally
sudo mv ./bin/act /usr/local/bin/act
```

### Basic Commands

```bash
# List all available workflows and jobs
act -l

# Run specific job from test.yml
act -j lint      # Run linting checks
act -j test      # Run tests with MySQL service
act -j build     # Run build job

# Run all jobs in a workflow
act                      # Run all jobs
act -W .github/workflows/test.yml  # Run specific workflow file

# Simulate specific event
act pull_request  # Simulate PR event
act push         # Simulate push event

# View logs in real-time
act -v           # Verbose output
act -j test -v   # Verbose for specific job
```

## Your Workflows

### 1. **test.yml** - Lint & Test
- **Triggers**: Pull requests and pushes to main/develop
- **Jobs**:
  - `lint`: Runs golangci-lint and go vet
  - `test`: Runs unit tests with MySQL 8.0 service
  - `build`: Builds binaries to bin/

Run locally:
```bash
act -j lint                    # Check code quality
act -j test                    # Run full test suite
./bin/act push -W .github/workflows/test.yml  # Simulate push event
```

### 2. **build.yml** - Build & Push Docker
- **Trigger**: Pushes to main/develop
- **Job**: `build-and-push` - Builds Docker images

Run locally:
```bash
act -j build-and-push
```

### 3. **deploy-local.yml** - Deploy to Kind
- **Trigger**: Manual workflow dispatch
- **Job**: `deploy-kind` - Deploys to local Kubernetes

Run locally:
```bash
act workflow_dispatch -W .github/workflows/deploy-local.yml
```

### 4. **deploy-aws.yml** - Deploy to EKS
- **Trigger**: Manual workflow dispatch
- **Job**: `deploy-eks` - Deploys to AWS EKS

## Troubleshooting

### Docker not running
```bash
systemctl status docker
sudo systemctl start docker
```

### Out of memory errors
```bash
act -j test --container-architecture linux/amd64 -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:full-latest
```

### Test database connection issues
```bash
# MySQL service is automatically started by act
# Verify connectivity in test output
act -j test -v
```

### Secrets not available locally
- Create a `.secrets` file in repo root (add to .gitignore)
- Format: `KEY=VALUE`
- Run: `act -j test --secret-file .secrets`

## Configuration

The `.actrc` file contains default settings:
- Uses full Ubuntu image for compatibility
- Sets container architecture to linux/amd64
- Enables verbose output by default

Edit `.actrc` to customize behavior without CLI flags.

## Performance Tips

1. **First run is slow** (downloads Docker images)
2. **Cache builds**: Images are cached after first run
3. **Selective testing**: Use `-j jobname` instead of running everything
4. **Skip unavailable jobs**: Some workflows (AWS deploy) need credentials

## CI/CD Workflow Example

```bash
# Before pushing to GitHub:
act -j lint        # ✅ Linting passes
act -j test        # ✅ Tests pass  
act -j build       # ✅ Docker builds successfully

# Then push
git add .
git commit -m "feature: xyz"
git push origin feature-branch
```

## references
- **act documentation**: https://github.com/nektos/act
- **GitHub Actions docs**: https://docs.github.com/en/actions
- **Your workflows**: `.github/workflows/`
