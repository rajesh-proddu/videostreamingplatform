#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
METADATA_URL="http://localhost:8080"
DATA_URL="http://localhost:8081"
TEST_FILE="test-video.bin"
TEST_FILE_SIZE=$((10 * 1024 * 1024)) # 10MB for testing
CHUNK_SIZE=$((5 * 1024 * 1024)) # 5MB chunks

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if services are running
check_services() {
    log_info "Checking if services are running..."
    
    # Check metadata service
    if ! curl -s "${METADATA_URL}/health" > /dev/null 2>&1; then
        log_error "Metadata service not responding at ${METADATA_URL}"
        return 1
    fi
    log_info "✓ Metadata service is healthy"
    
    # Check data service
    if ! curl -s "${DATA_URL}/health" > /dev/null 2>&1; then
        log_error "Data service not responding at ${DATA_URL}"
        return 1
    fi
    log_info "✓ Data service is healthy"
}

# Start services
start_services() {
    log_info "Starting services..."
    
    # Start in background
    cd /home/rajesh/go_workspace/videostreamingplatform
    ./build/metadata-service &
    METADATA_PID=$!
    
    ./build/data-service &
    DATA_PID=$!
    
    # Wait for services to start
    sleep 2
    
    # Give them more time to initialize
    for i in {1..30}; do
        if check_services; then
            log_info "Services started successfully"
            return 0
        fi
        sleep 1
    done
    
    log_error "Services failed to start"
    return 1
}

# Create test file
create_test_file() {
    log_info "Creating test file (${TEST_FILE_SIZE} bytes)..."
    dd if=/dev/urandom of="${TEST_FILE}" bs=1024 count=$((TEST_FILE_SIZE / 1024)) 2>/dev/null
    log_info "✓ Test file created: ${TEST_FILE}"
}

# Calculate checksum
calculate_checksum() {
    md5sum "$1" | awk '{print $1}'
}

# Create video metadata
create_video() {
    log_info "Creating video metadata..."
    
    RESPONSE=$(curl -s -X POST "${METADATA_URL}/videos" \
        -H "Content-Type: application/json" \
        -d '{
            "title": "Test Video",
            "description": "HTTP/2 Streaming Test",
            "duration_seconds": 300,
            "file_size_bytes": '${TEST_FILE_SIZE}',
            "user_id": "test-user-123"
        }')
    
    VIDEO_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -z "$VIDEO_ID" ]; then
        log_error "Failed to create video. Response: $RESPONSE"
        return 1
    fi
    
    log_info "✓ Video created with ID: ${VIDEO_ID}"
}

# Initiate upload
initiate_upload() {
    log_info "Initiating upload session..."
    
    RESPONSE=$(curl -s -X POST "${DATA_URL}/uploads/initiate" \
        -H "Content-Type: application/json" \
        -d '{
            "video_id": "'${VIDEO_ID}'",
            "user_id": "test-user-123",
            "total_size": '${TEST_FILE_SIZE}'
        }')
    
    UPLOAD_ID=$(echo "$RESPONSE" | grep -o '"upload_id":"[^"]*"' | cut -d'"' -f4)
    CHUNK_SIZE=$(echo "$RESPONSE" | grep -o '"chunk_size":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$UPLOAD_ID" ]; then
        log_error "Failed to initiate upload. Response: $RESPONSE"
        return 1
    fi
    
    log_info "✓ Upload session initiated: ${UPLOAD_ID}"
    log_info "  Chunk size: ${CHUNK_SIZE} bytes"
}

# Upload chunks
upload_chunks() {
    log_info "Uploading chunks..."
    
    local uploaded=0
    local chunk_num=0
    local total_chunks=$(( (TEST_FILE_SIZE + CHUNK_SIZE - 1) / CHUNK_SIZE ))
    
    while [ $uploaded -lt $TEST_FILE_SIZE ]; do
        local remaining=$((TEST_FILE_SIZE - uploaded))
        local to_read=$CHUNK_SIZE
        
        if [ $remaining -lt $CHUNK_SIZE ]; then
            to_read=$remaining
        fi
        
        # Extract chunk from test file
        dd if="${TEST_FILE}" of="chunk-${chunk_num}.bin" bs=1 skip=$uploaded count=$to_read 2>/dev/null
        
        # Calculate checksum
        CHECKSUM=$(calculate_checksum "chunk-${chunk_num}.bin")
        
        # Upload chunk
        log_info "  Uploading chunk $((chunk_num + 1))/${total_chunks} ($(( to_read / 1024 ))KB)..."
        
        RESPONSE=$(curl -s -X POST "${DATA_URL}/uploads/${UPLOAD_ID}/chunks?chunkIndex=${chunk_num}" \
            -H "Content-Type: application/octet-stream" \
            --data-binary "@chunk-${chunk_num}.bin")
        
        # Check if upload was successful
        if echo "$RESPONSE" | grep -q '"status":"success"'; then
            log_info "    ✓ Chunk uploaded"
        else
            log_error "    Failed to upload chunk. Response: $RESPONSE"
            return 1
        fi
        
        # Clean up chunk file
        rm "chunk-${chunk_num}.bin"
        
        uploaded=$((uploaded + to_read))
        chunk_num=$((chunk_num + 1))
        
        # Show progress
        PROGRESS=$(curl -s "${DATA_URL}/uploads/${UPLOAD_ID}/progress")
        PERCENTAGE=$(echo "$PROGRESS" | grep -o '"percentage":[^,}]*' | cut -d':' -f2)
        SPEED=$(echo "$PROGRESS" | grep -o '"speed_mbps":[^,}]*' | cut -d':' -f2)
        
        if [ ! -z "$PERCENTAGE" ]; then
            log_info "    Progress: ${PERCENTAGE}% (Speed: ${SPEED} Mbps)"
        fi
    done
    
    log_info "✓ All chunks uploaded"
}

# Complete upload
complete_upload() {
    log_info "Completing upload..."
    
    RESPONSE=$(curl -s -X POST "${DATA_URL}/uploads/${UPLOAD_ID}/complete" \
        -H "Content-Type: application/json")
    
    if echo "$RESPONSE" | grep -q '"status":"completed"'; then
        log_info "✓ Upload completed successfully"
    else
        log_error "Failed to complete upload. Response: $RESPONSE"
        return 1
    fi
}

# Download file
download_file() {
    log_info "Downloading file..."
    
    curl -s -o "downloaded-${VIDEO_ID}.bin" "${DATA_URL}/videos/${VIDEO_ID}/download"
    
    if [ -f "downloaded-${VIDEO_ID}.bin" ]; then
        log_info "✓ File downloaded: downloaded-${VIDEO_ID}.bin"
    else
        log_error "Failed to download file"
        return 1
    fi
}

# Verify checksum
verify_checksum() {
    log_info "Verifying checksum..."
    
    ORIGINAL_CHECKSUM=$(calculate_checksum "${TEST_FILE}")
    DOWNLOADED_CHECKSUM=$(calculate_checksum "downloaded-${VIDEO_ID}.bin")
    
    if [ "$ORIGINAL_CHECKSUM" = "$DOWNLOADED_CHECKSUM" ]; then
        log_info "✓ Checksums match!"
        log_info "  Original:  ${ORIGINAL_CHECKSUM}"
        log_info "  Downloaded: ${DOWNLOADED_CHECKSUM}"
    else
        log_error "Checksum mismatch!"
        log_error "  Original:  ${ORIGINAL_CHECKSUM}"
        log_error "  Downloaded: ${DOWNLOADED_CHECKSUM}"
        return 1
    fi
}

# Cleanup
cleanup() {
    log_info "Cleaning up..."
    
    # Kill services if we started them
    if [ ! -z "$METADATA_PID" ]; then
        kill $METADATA_PID 2>/dev/null || true
    fi
    if [ ! -z "$DATA_PID" ]; then
        kill $DATA_PID 2>/dev/null || true
    fi
    
    # Remove test files
    rm -f "${TEST_FILE}" "chunk-"*.bin "downloaded-"*.bin
}

# Main execution
main() {
    log_info "Starting HTTP/2 Streaming Test"
    log_info "================================"
    
    # Cleanup on exit
    trap cleanup EXIT
    
    # Only start services if they're not already running
    if ! check_services; then
        start_services || exit 1
    else
        log_info "Services already running"
    fi
    
    create_test_file || exit 1
    create_video || exit 1
    initiate_upload || exit 1
    upload_chunks || exit 1
    complete_upload || exit 1
    download_file || exit 1
    verify_checksum || exit 1
    
    log_info ""
    log_info "✅ All tests passed!"
    log_info "================================"
}

main "$@"
