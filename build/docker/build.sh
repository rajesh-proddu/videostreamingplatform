#!/bin/bash
set -e

# Build script for video streaming platform
# Builds Docker images for all services

REGISTRY=${REGISTRY:-videostreamingplatform}
VERSION=${VERSION:-latest}
BUILD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "Building Docker images..."
echo "Registry: $REGISTRY"
echo "Version: $VERSION"

# Build metadata service
echo "Building metadataservice..."
docker build -f "$BUILD_DIR/build/docker/metadataservice.Dockerfile" \
  -t "$REGISTRY/metadataservice:$VERSION" \
  -t "$REGISTRY/metadataservice:latest" \
  "$BUILD_DIR"

# Build data service
echo "Building dataservice..."
docker build -f "$BUILD_DIR/build/docker/dataservice.Dockerfile" \
  -t "$REGISTRY/dataservice:$VERSION" \
  -t "$REGISTRY/dataservice:latest" \
  "$BUILD_DIR"

echo "Docker build completed successfully!"
echo "Images:"
echo "  $REGISTRY/metadataservice:$VERSION"
echo "  $REGISTRY/dataservice:$VERSION"
docker images | grep "$REGISTRY"
