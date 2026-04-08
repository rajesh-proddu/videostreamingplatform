#!/bin/bash

# Simple integration test for HTTP/2 streaming
set -e

METADATA_URL="http://localhost:8080"
DATA_URL="http://localhost:8081"

echo "=== Starting HTTP/2 Streaming Integration Test ==="
echo ""

# Start services
echo "[1/5] Starting services..."
cd /home/rajesh/go_workspace/videostreamingplatform

# Kill any existing processes
pkill -f "metadata-service\|data-service" || true
sleep 1

# Start services
./build/metadata-service > /tmp/metadata.log 2>&1 &
METADATA_PID=$!
echo "  Metadata service PID: $METADATA_PID"

./build/data-service > /tmp/data.log 2>&1 &
DATA_PID=$!
echo "  Data service PID: $DATA_PID"

# Wait for services to start
echo "  Waiting for services to be healthy..."
for i in {1..30}; do
    if curl -s "${METADATA_URL}/health" > /dev/null && curl -s "${DATA_URL}/health" > /dev/null; then
        echo "  ✓ Services are healthy"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "  ✗ Services failed to start"
        cat /tmp/metadata.log
        cat /tmp/data.log
        exit 1
    fi
    sleep 1
done
echo ""

# Test health endpoints
echo "[2/5] Testing health endpoints..."
echo -n "  Metadata health: "
curl -s "${METADATA_URL}/health" && echo ""
echo -n "  Data health: "
curl -s "${DATA_URL}/health" && echo ""
echo ""

# Create test video
echo "[3/5] Creating test video..."
VIDEO_RESPONSE=$(curl -s -X POST "${METADATA_URL}/videos" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Test Stream",
        "description": "HTTP/2 streaming test",
        "duration_seconds": 300,
        "file_size_bytes": 5242880,
        "user_id": "test-user"
    }' 2>/dev/null || echo "{}")

VIDEO_ID=$(echo "$VIDEO_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")

if [ -z "$VIDEO_ID" ]; then
    echo "  ✗ Failed to create video"
    echo "  Response: $VIDEO_RESPONSE"
    kill $METADATA_PID $DATA_PID
    exit 1
fi

echo "  ✓ Video created: $VIDEO_ID"
echo ""

# Test upload initiation
echo "[4/5] Testing upload initiation..."
UPLOAD_RESPONSE=$(curl -s -X POST "${DATA_URL}/uploads/initiate" \
    -H "Content-Type: application/json" \
    -d '{
        "video_id": "'${VIDEO_ID}'",
        "user_id": "test-user",
        "total_size": 5242880
    }' 2>/dev/null || echo "{}")

UPLOAD_ID=$(echo "$UPLOAD_RESPONSE" | grep -o '"upload_id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$UPLOAD_ID" ]; then
    echo "  ✗ Failed to initiate upload"
    echo "  Response: $UPLOAD_RESPONSE"
    kill $METADATA_PID $DATA_PID
    exit 1
fi

echo "  ✓ Upload initiated: $UPLOAD_ID"
echo ""

# Test progress endpoint
echo "[5/5] Testing progress endpoint..."
PROGRESS_RESPONSE=$(curl -s "${DATA_URL}/uploads/${UPLOAD_ID}/progress" 2>/dev/null || echo "{}")

PERCENTAGE=$(echo "$PROGRESS_RESPONSE" | grep -o '"percentage":[^,}]*' | cut -d':' -f2)

if [ -z "$PERCENTAGE" ]; then
    echo "  ✗ Failed to get progress"
    echo "  Response: $PROGRESS_RESPONSE"
    kill $METADATA_PID $DATA_PID
    exit 1
fi

echo "  ✓ Progress retrieved: ${PERCENTAGE}%"
echo ""

# Cleanup
echo "=== Test Summary ==="
echo "✓ Health endpoints working"
echo "✓ Video creation working"
echo "✓ Upload initiation working"
echo "✓ Progress tracking working"
echo ""
echo "Cleaning up..."

kill $METADATA_PID $DATA_PID 2>/dev/null || true
sleep 1

echo "✅ All tests passed!"
