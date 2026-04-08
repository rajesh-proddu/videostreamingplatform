# GitHub Actions Local Runner

This directory contains everything needed to run GitHub workflows locally before pushing to GitHub.

## Directory Structure

```
rungithublocal/
├── bin/
│   └── act              # GitHub Actions runner binary
├── run-workflow         # Helper script for easy workflow execution
├── .actrc              # act configuration file
└── README.md           # This file
```

## Quick Start

From the repository root:

```bash
# List all available workflows
./rungithublocal/run-workflow list

# Run specific workflow
./rungithublocal/run-workflow lint       # Code linting
./rungithublocal/run-workflow test       # Unit tests
./rungithublocal/run-workflow build      # Docker build
./rungithublocal/run-workflow test-all   # Run: lint → test → build
```

Or from this directory:

```bash
./run-workflow lint
```

## Available Commands

| Command | Description |
|---------|-------------|
| `list` | List all available workflows and jobs |
| `lint` | Run code linting checks |
| `test` | Run full test suite with MySQL |
| `build` | Build Docker images |
| `test-all` | Run complete CI pipeline (lint → test → build) |
| `deploy-local` | Deploy to local Kind cluster |
| `deploy-aws` | Deploy to AWS EKS (requires credentials) |

## Options

- `-v, --verbose` - Show detailed output
- `-h, --help` - Show help message

## Examples

```bash
# From repo root, run full CI pipeline
./rungithublocal/run-workflow test-all

# Run with verbose output
./rungithublocal/run-workflow test --verbose

# List all jobs before running
./rungithublocal/run-workflow list
```

## Files

### `bin/act`
The act binary (v0.2.87) - runs GitHub Actions workflows locally using Docker.

### `run-workflow`
Helper script with user-friendly interface. Handles:
- Validating Docker is running
- Checking that act is installed
- Providing colored output
- Running workflows with proper arguments

### `.actrc`
Configuration file for act. Settings:
- Uses full Ubuntu image for compatibility
- Container architecture: linux/amd64
- Verbose output by default

## Prerequisites

- Docker (must be running)
- Bash shell

## Troubleshooting

### Docker not running
```bash
sudo systemctl start docker
```

### Permission denied
```bash
chmod +x run-workflow
chmod +x bin/act
```

### Slow first run
- Normal! Docker images are downloaded on first execution
- Subsequent runs use cached images

## Documentation

For comprehensive information, see: `docs/LOCAL_WORKFLOWS.md`

## References

- **act**: https://github.com/nektos/act
- **GitHub Actions**: https://docs.github.com/en/actions
