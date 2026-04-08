# Binary Build and Cleanup Guide

This guide explains how binaries are managed in the Video Streaming Platform project.

## Binary Locations

### Build Output Location
All compiled binaries are ONLY created in the `bin/` folder at the repository root:

```
videostreamingplatform/
├── bin/
│   ├── .gitkeep              # Ensures bin/ folder is tracked in git
│   ├── dataservice           # Generated after build
│   └── metadataservice       # Generated after build
```

### Why Only `bin/` Folder?
1. **Clean repository** - No binaries scattered across the project
2. **Easy cleanup** - Simple to delete all binaries at once
3. **Clear separation** - Build artifacts separated from source code
4. **Easier gitignore** - Single rule `bin/` ignores all binaries while `.gitkeep` ensures folder exists

## Building Binaries

### Standard Build
```bash
# From project root
make -f build/Makefile build

# Or from build directory
cd build
make build
```

Output:
```
Building dataservice...
CGO_ENABLED=0 GOOS=linux go build -o bin/dataservice ./dataservice
Building metadataservice...
CGO_ENABLED=0 GOOS=linux go build -o bin/metadataservice ./metadataservice
✓ All services built
```

### Build Individual Services
```bash
make -f build/Makefile build-dataservice
make -f build/Makefile build-metadataservice
```

### Manual Build (without Makefile)
```bash
# From repository root
go build -o bin/dataservice ./dataservice
go build -o bin/metadataservice ./metadataservice
```

## Binary Cleanup

### Using Cleanup Script (Recommended)
The dedicated cleanup script removes all binaries and build artifacts:

```bash
# Basic cleanup (binaries only)
bash build/scripts/cleanup.sh

# With Docker cleanup
bash build/scripts/cleanup.sh --docker

# Aggressive cleanup (includes Docker system prune)
bash build/scripts/cleanup.sh --docker-aggressive
```

### Using Makefile
```bash
# Clean binaries and coverage files
make -f build/Makefile clean

# Clean only binaries
make -f build/Makefile clean-bin

# Clean only coverage files  
make -f build/Makefile clean-coverage

# Clean only Docker artifacts
make -f build/Makefile clean-docker

# Clean everything
make -f build/Makefile clean-all
```

### Manual Cleanup
```bash
# Remove all binaries
rm -rf bin/*

# Keep .gitkeep for git tracking
echo "" > bin/.gitkeep

# Remove test binaries
rm -f test-*

# Remove coverage files
rm -f coverage*.out
```

## Running Binaries

### Using Built Binaries
```bash
# Data Service (HTTP/2 + gRPC)
./bin/dataservice

# Metadata Service (HTTP REST)
./bin/metadataservice
```

### With Configuration
```bash
# Set environment variables before running
export HTTP_PORT=8081
export GRPC_PORT=50051
export ENVIRONMENT=dev
./bin/dataservice

# For metadata service
export HTTP_PORT=8080
export MYSQL_HOST=localhost
./bin/metadataservice
```

### Running from Docker Compose
```bash
# Uses bin/ folder from build/docker/Dockerfile referencing bin/
docker-compose -f build/docker-compose.yml up -d
```

## Directory Structure for Binaries

```
Project Root
├── bin/                      # ONLY location for binaries
│   ├── .gitkeep             # Empty file to track folder in git
│   ├── dataservice          # Built from ./dataservice
│   └── metadataservice      # Built from ./metadataservice
├── build/
│   ├── Makefile             # Build automation
│   ├── docker/
│   │   ├── dataservice.Dockerfile
│   │   ├── metadataservice.Dockerfile
│   │   └── build.sh
│   └── scripts/
│       └── cleanup.sh       # Cleanup script
├── dataservice/             # Source code only (no binaries)
├── metadataservice/         # Source code only (no binaries)
└── .gitignore
    └── Contains: bin/       # Ignore bin contents but not .gitkeep
```

## Git Tracking

### What Gets Committed
- Source code (`.go` files)
- Configuration (`go.mod`, `go.sum`, `.env.example`)
- Build configuration (`Makefile`, `Dockerfile`)
- Documentation (`README.md`, docs/`)
- `.gitkeep` in `bin/` folder

### What Gets Ignored
- All binary files in `bin/`
- Coverage reports (`coverage*.out`)
- Test artifacts
- Temporary files

### .gitignore Configuration
```
# Binaries
*.exe
*.exe~
*.dll
*.so
*.so.*
*.dylib

# Build outputs
bin/              # Ignore everything in bin/
coverage.out
*.test
```

The `.gitkeep` file in `bin/` is NOT ignored, so the folder structure is preserved.

## Verification Steps

### Verify Binaries Only in bin/
```bash
# Should show only .gitkeep
ls -la bin/

# Should show no stray binaries
find . -type f -executable ! -path "*/.*" ! -path "*/bin/*" ! -name "*.sh"
```

### Verify Build Output
```bash
# Build should succeed
make -f build/Makefile build

# Check binaries exist
file bin/dataservice
file bin/metadataservice
```

### Verify Cleanup
```bash
# Run cleanup
bash build/scripts/cleanup.sh

# Verify empty except .gitkeep
ls -la bin/
```

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Build binaries
  run: make -f build/Makefile build

- name: Verify binaries in bin/
  run: ls -la bin/

- name: Cleanup before deploy
  run: bash build/scripts/cleanup.sh

- name: Build Docker images
  run: bash build/docker/build.sh

- name: Deploy
  run: kubectl apply -f build/scripts/k8s/
```

## Common Issues & Solutions

### Issue: Binaries appear in wrong location
**Solution**: Verify Makefile has correct output paths
```makefile
build-dataservice:
	go build -o bin/dataservice ./dataservice  # ← Must be bin/
```

### Issue: Cleanup script doesn't exist
**Solution**: Make sure script has execute permissions
```bash
chmod +x build/scripts/cleanup.sh
```

### Issue: Can't rebuild after cleanup
**Solution**: Recreate `bin/.gitkeep` if needed
```bash
mkdir -p bin
touch bin/.gitkeep
make -f build/Makefile build
```

### Issue: Git tracking bin/ folder
**Solution**: The `.gitkeep` file ensures folder is tracked
```bash
# Verify .gitkeep exists
ls -la bin/.gitkeep

# If missing, recreate
touch bin/.gitkeep
git add bin/.gitkeep
git commit -m "Track bin folder structure"
```

## Development Workflow

### Step 1: Clean Previous Build
```bash
bash build/scripts/cleanup.sh
```

### Step 2: Build New Binaries
```bash
make -f build/Makefile build
```

### Step 3: Test Binaries
```bash
# Test with configuration
export ENVIRONMENT=dev
./bin/dataservice &
./bin/metadataservice &

# Verify in another terminal
curl http://localhost:8080/health
curl http://localhost:8081/health
```

### Step 4: Deploy
```bash
# Build Docker images
bash build/docker/build.sh

# Deploy to Kubernetes
make -f build/Makefile deploy-local
```

### Step 5: Cleanup
```bash
# Before committing
bash build/scripts/cleanup.sh

# Verify only source files are tracked
git status
```

## Best Practices

✅ **DO**:
- Always build to `bin/` folder
- Use cleanup script before commits
- Keep `.gitkeep` in `bin/`
- Update Makefile if paths change
- Document any new build outputs

❌ **DON'T**:
- Create binaries elsewhere (no `./dataservice/bin/`, etc.)
- Commit binary files to git
- Remove `.gitkeep` without reason
- Mix build outputs with source code
- Ignore the cleanup script

## Quick Reference

```bash
# Build
make -f build/Makefile build

# Test  
make -f build/Makefile test

# Cleanup
bash build/scripts/cleanup.sh

# Build Docker
bash build/docker/build.sh

# Deploy
make -f build/Makefile deploy-local

# Full workflow
bash build/scripts/cleanup.sh && \
make -f build/Makefile build && \
make -f build/Makefile test && \
bash build/docker/build.sh
```
