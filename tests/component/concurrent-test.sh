#!/bin/bash
# Concurrent Transfer Test - Simulates multiple parallel uploads

set -e

METADATA_URL="http://localhost:8080"
DATA_URL="http://localhost:8081"

echo "=== Concurrent Transfer Test ==="
echo ""
echo "Configuration:"
echo "  - Number of concurrent uploads: 5"
echo "  - Test duration: 60 seconds"
echo "  - Monitor frequency: 10 seconds"
echo ""

# Start services in background
echo "Starting services..."
cd /home/rajesh/go_workspace/videostreamingplatform

pkill -f "metadata-service\|data-service" || true
sleep 1

./build/metadata-service > /tmp/metadata.log 2>&1 &
METADATA_PID=$!

./build/data-service > /tmp/data.log 2>&1 &
DATA_PID=$!

# Wait for services
echo "Waiting for services..."
for i in {1..30}; do
    if curl -s "${METADATA_URL}/health" > /dev/null && curl -s "${DATA_URL}/health" > /dev/null; then
        echo "✓ Services healthy"
        break
    fi
    sleep 1
done

# Function to create and track upload
upload_file() {
    local upload_num=$1
    local file_size=$((5 * 1024 * 1024)) # 5MB for testing
    
    # Create video
    video_resp=$(curl -s -X POST "${METADATA_URL}/videos" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Concurrent Test Video $upload_num\",
            \"description\": \"Concurrent transfer test\",
            \"duration_seconds\": 60,
            \"file_size_bytes\": $file_size,
            \"user_id\": \"test-user-$upload_num\"
        }")
    
    video_id=$(echo "$video_resp" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "Upload $upload_num: Video ID: $video_id"
    
    # Initiate upload
    upload_resp=$(curl -s -X POST "${DATA_URL}/uploads/initiate" \
        -H "Content-Type: application/json" \
        -d "{
            \"video_id\": \"$video_id\",
            \"user_id\": \"test-user-$upload_num\",
            \"total_size\": $file_size
        }")
    
    upload_id=$(echo "$upload_resp" | grep -o '"upload_id":"[^"]*"' | cut -d'"' -f4)
    echo "Upload $upload_num: Upload ID: $upload_id"
    
    # Store for progress tracking
    echo "$upload_id" > /tmp/upload_$upload_num.id
}

# Start 5 concurrent uploads
echo ""
echo "Starting 5 concurrent uploads..."
for i in {1..5}; do
    upload_file $i &
done

wait

echo ""
echo "All uploads initiated. Monitoring progress..."
echo ""

# Monitor progress for 30 seconds
start_time=$(date +%s)
end_time=$((start_time + 60))

while [ $(date +%s) -lt $end_time ]; do
    echo "--- Progress Report ($(date +%T)) ---"
    
    for i in {1..5}; do
        if [ -f /tmp/upload_$i.id ]; then
            upload_id=$(cat /tmp/upload_$i.id)
            progress=$(curl -s "${DATA_URL}/uploads/${upload_id}/progress" 2>/dev/null || echo "{}")
            
            percentage=$(echo "$progress" | grep -o '"percentage":[^,}]*' | cut -d':' -f2)
            speed=$(echo "$progress" | grep -o '"speed_mbps":[^,}]*' | cut -d':' -f2)
            uploaded=$(echo "$progress" | grep -o '"uploaded_bytes":[^,}]*' | cut -d':' -f2)
            
            if [ ! -z "$percentage" ]; then
                printf "  Upload %d: %.1f%% | Speed: %.2f Mbps | Bytes: %s\n" \
                    $i $percentage $speed $uploaded
            fi
        fi
    done
    
    echo ""
    sleep 10
done

# Cleanup
echo ""
echo "Cleaning up..."
kill $METADATA_PID $DATA_PID 2>/dev/null || true
rm -f /tmp/upload_*.id

echo "✅ Concurrent transfer test completed"
