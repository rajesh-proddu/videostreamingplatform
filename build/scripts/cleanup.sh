#!/bin/bash

# Cleanup script for video streaming platform binaries and build artifacts
# This script removes all compiled binaries and build artifacts

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Counter for deleted files
deleted_count=0

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Video Streaming Platform Cleanup${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Function to remove directory
remove_dir() {
	local target="$1"
	local description="$2"
	
	if [ -d "$target" ]; then
		echo -e "${YELLOW}Removing ${description}...${NC}"
		find "$target" -type f -exec rm -f {} \;
		rmdir "$target" 2>/dev/null || true
		deleted_count=$((deleted_count + 1))
		echo -e "${GREEN}✓ Cleaned: ${description}${NC}"
	fi
}

# Function to remove file pattern
remove_files() {
	local pattern="$1"
	local description="$2"
	
	if find "$SCRIPT_DIR" -maxdepth 2 -name "$pattern" -type f 2>/dev/null | grep -q .; then
		echo -e "${YELLOW}Removing ${description}...${NC}"
		find "$SCRIPT_DIR" -maxdepth 2 -name "$pattern" -type f -exec rm -f {} \;
		deleted_count=$((deleted_count + 1))
		echo -e "${GREEN}✓ Cleaned: ${description}${NC}"
	fi
}

# Cleanup bin directory
if [ -d "$SCRIPT_DIR/bin" ]; then
	echo -e "${YELLOW}Removing binary files...${NC}"
	rm -rf "$SCRIPT_DIR/bin"/*
	deleted_count=$((deleted_count + 1))
	echo -e "${GREEN}✓ Cleaned: bin/ directory${NC}"
fi

# Cleanup coverage files
echo -e "${YELLOW}Removing coverage files...${NC}"
find "$SCRIPT_DIR" -maxdepth 1 -name "coverage*.out" -type f -exec rm -f {} \;
deleted_count=$((deleted_count + 1))
echo -e "${GREEN}✓ Cleaned: Coverage files${NC}"

# Cleanup test binaries
echo -e "${YELLOW}Removing test binaries...${NC}"
find "$SCRIPT_DIR" -maxdepth 1 -name "test-*" -type f -exec rm -f {} \;
echo -e "${GREEN}✓ Cleaned: Test binaries${NC}"

# Cleanup dataservice/bin if exists
if [ -d "$SCRIPT_DIR/dataservice/bin" ]; then
	echo -e "${YELLOW}Removing dataservice/bin...${NC}"
	rm -rf "$SCRIPT_DIR/dataservice/bin"
	echo -e "${GREEN}✓ Cleaned: dataservice/bin${NC}"
fi

# Optional: Clean Docker images
if [ "$1" == "--docker" ] || [ "$1" == "-d" ]; then
	echo ""
	echo -e "${YELLOW}Removing Docker images and containers...${NC}"
	
	# Stop and remove videostreamingplatform containers
	docker-compose -f "$SCRIPT_DIR/build/docker-compose.yml" down --remove-orphans 2>/dev/null || true
	
	# Remove images
	docker rmi videostreamingplatform/dataservice:latest 2>/dev/null || true
	docker rmi videostreamingplatform/metadataservice:latest 2>/dev/null || true
	
	# Optionally prune all
	if [ "$1" == "--docker-aggressive" ] || [ "$1" == "-da" ]; then
		echo -e "${YELLOW}Aggressive Docker cleanup (system prune)...${NC}"
		docker system prune -f --volumes
	fi
	
	echo -e "${GREEN}✓ Cleaned: Docker artifacts${NC}"
fi

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Cleanup Summary:${NC}"
echo -e "${GREEN}✓ Removed all binary files from bin/${NC}"
echo -e "${GREEN}✓ Removed test binaries${NC}"
echo -e "${GREEN}✓ Removed coverage reports${NC}"
if [ "$1" == "--docker" ] || [ "$1" == "-d" ] || [ "$1" == "--docker-aggressive" ] || [ "$1" == "-da" ]; then
	echo -e "${GREEN}✓ Removed Docker artifacts${NC}"
fi
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "${GREEN}Ready for fresh build!${NC}"
echo ""
echo "To rebuild:"
echo "  make -f build/Makefile build"
echo ""
echo "To rebuild Docker images:"
echo "  bash build/docker/build.sh"
echo ""
