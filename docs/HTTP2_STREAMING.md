# HTTP/2 Streaming Implementation

## Overview

This document describes the HTTP/2 streaming implementation for chunked file upload and download with resume capability, retry logic, and real-time progress tracking.

## Architecture

### Components

#### 1. Streaming Package (`internal/streaming/`)
Core streaming functionality with:
- **UploadSession**: Tracks ongoing uploads, manages chunk reception, calculates metadata
- **ChunkReader**: Reads files in chunks with size validation
- **ChunkWriter**: Writes chunks out-of-order with offset mapping
- **ChunkMetadata**: Holds metadata about individual chunks (offset, size, MD5, etc.)

#### 2. Retry Package (`internal/retry/`)
Automatic retry mechanism with:
- **Exponential Backoff**: Base × Multiplier^attempt with jitter
- **Context-Aware**: Respects cancellation and deadlines
- **Temporary Error Detection**: Distinguishes between temporary and permanent errors
- **Configuration**: Customizable max retries, backoff timings, and jitter

#### 3. Progress Package (`internal/progress/`)
Real-time upload/download tracking:
- **Atomic Operations**: Thread-safe updates without locks on hot path
- **Speed Calculation**: Dynamic B/s calculation based on time elapsed
- **ETA**: Estimated time to completion based on current speed
- **Formatted Output**: Ready-to-display strings for UI integration

#### 4. Streaming Handlers (`internal/handlers/streaming.go`)
HTTP handlers for the streaming API:
- **InitiateUpload**: Creates upload session, returns upload ID and chunk size
- **UploadChunk**: Uploads individual chunks with automatic retry to S3
- **GetUploadProgress**: Returns real-time progress with speed and ETA
- **CompleteUpload**: Validates all chunks received, finalizes upload
- **StreamDownload**: Streams download with automatic retry and progress tracking

### Data Flow

```
┌─────────────────────────────────────────────────┐
│         HTTP/2 Client                           │
└────────────┬────────────────────────────────────┘
             │
             ├─► POST /uploads/initiate           (1)
             ├─► POST /uploads/{id}/chunks        (2-N)
             ├─► GET /uploads/{id}/progress       (polling)
             ├─► POST /uploads/{id}/complete      (N+1)
             └─► GET /videos/{id}/download        (download)
             │
┌────────────▼────────────────────────────────────┐
│    HTTP/2 Data Service (Port 8081)               │
│  ┌──────────────────────────────────────┐        │
│  │ Streaming Handlers                    │        │
│  │ - InitiateUpload                      │        │
│  │ - UploadChunk (+ Retry logic)         │        │
│  │ - GetUploadProgress                   │        │
│  │ - CompleteUpload                      │        │
│  │ - StreamDownload (+ Retry logic)      │        │
│  └──────────────────────────────────────┘        │
│            │         │         │                  │
│  ┌─────────▼─┐   ┌──▼──────┐  │                   │
│  │ S3 Client │   │ MySQL   │  │                   │
│  │ (with     │   │ (with   │  │                   │
│  │ retry)    │   │ metadata│  │                   │
│  └───────────┘   └─────────┘  │                   │
│                                │                   │
│  ┌──────────────────────────┐  │                   │
│  │ Progress/Streaming       │  │                   │
│  │ Utilities                │◄─┤                   │
│  └──────────────────────────┘  │                   │
└────────────────────────────────┘
```

## API Endpoints

### 1. Initiate Upload
**POST** `/uploads/initiate`

Request:
```json
{
  "video_id": "uuid",
  "user_id": "uuid",
  "total_size": 104857600
}
```

Response:
```json
{
  "upload_id": "uuid",
  "chunk_size": 5242880,
  "message": "Upload session initiated successfully"
}
```

### 2. Upload Chunk
**POST** `/uploads/{uploadId}/chunks?chunkIndex={index}`

Binary request body with chunk data.

Response:
```json
{
  "status": "Success",
  "message": "Chunk 0 received"
}
```

### 3. Get Upload Progress
**GET** `/uploads/{uploadId}/progress`

Response:
```json
{
  "upload_id": "uuid",
  "percentage": 25.5,
  "uploaded_bytes": 26843545,
  "total_bytes": 104857600,
  "uploaded_chunks": 5,
  "total_chunks": 20,
  "speed_mbps": 2.5,
  "estimated_seconds": 315
}
```

### 4. Complete Upload
**POST** `/uploads/{uploadId}/complete`

Request: Empty or any format

Response:
```json
{
  "status": "Upload completed successfully",
  "video_id": "uuid"
}
```

Error response (if chunks missing):
```json
{
  "error": "Upload incomplete",
  "missing_chunks": [3, 7, 15]
}
```

### 5. Stream Download
**GET** `/videos/{id}/download`

Response: Binary video data with streaming headers
```
Content-Type: video/mp4
Content-Length: 104857600
Content-Disposition: attachment; filename=video.mp4
Cache-Control: no-cache, no-store, must-revalidate
```

## Configuration

### Retry Configuration
```go
config := retry.Config{
    MaxRetries:        3,
    InitialBackoff:    100 * time.Millisecond,
    MaxBackoff:        10 * time.Second,
    BackoffMultiplier: 2.0,
    JitterFraction:    0.1,
}
```

Default backoff schedule:
- Attempt 1: 100ms - 200ms (with jitter)
- Attempt 2: 200ms - 400ms (with jitter)
- Attempt 3: 400ms - 800ms (with jitter)

### Chunk Size Configuration
- Minimum: 1 MB
- Default: 5 MB
- Maximum: 100 MB

## Implementation Details

### Chunk Upload Process

1. **Client splits file** into 5MB chunks
2. **Initiates upload** → receives uploadID and confirms chunk size
3. **Uploads chunks** in parallel or sequential (configurable):
   - Each POST request sends one chunk
   - Retry automatically on failure
   - Progress tracked atomically
4. **Polls progress** → gets speed and ETA
5. **Completes upload** → validates all chunks received

### Out-of-Order Chunk Handling

The ChunkWriter supports out-of-order writes using offset mapping:
```
WriteChunk(offset, data) {
    chunks[offset] = data
    // Write all contiguous chunks from nextOffset
    for {
        if chunk exists at nextOffset {
            write(chunk)
            nextOffset += len(chunk)
        } else {
            break
        }
    }
}
```

This allows clients to resume uploads or use parallel uploads without ordering concerns.

### Retry Logic with Exponential Backoff

```
Execute(ctx, config, operation, isTemporary) {
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := operation(ctx, attempt)
        if err == nil {
            return nil
        }
        
        if !isTemporary(err) {
            return err  // Don't retry permanent errors
        }
        
        if attempt < maxRetries {
            backoff := calculateBackoff(attempt, config)
            select {
            case <-time.After(backoff):
                continue
            case <-ctx.Done():
                return ctx.Err()  // Ctx cancelled, stop retrying
            }
        }
    }
    return lastError
}
```

### Progress Tracking

Uses atomic operations to avoid contention on high-concurrency uploads:
```go
// Thread-safe increment of uploaded bytes
atomic.AddInt64(&tracker.uploadedBytes, int64(len(chunk)))

// Speed calculation
elapsed := time.Since(tracker.startTime)
speedBps := float64(tracker.uploadedBytes) / elapsed.Seconds()

// ETA calculation
remainingBytes := tracker.totalSize - tracker.uploadedBytes
estimatedSeconds := remainingBytes / int64(speedBps)
```

## Testing

### Integration Test
Basic end-to-end test verifying:
- Health endpoints responsive
- Video creation working
- Upload session initiation successful
- Progress tracking accessible

Run with:
```bash
./integration-test.sh
```

### Load Testing (Planned)
- 10 concurrent uploads
- Network failure scenarios
- Speed/ETA accuracy verification

### E2E Testing (Planned)
- Full upload/download cycle
- Chunk integrity verification
- Resume from partial upload
- Concurrent chunk uploads

## Error Handling

### Temporary Errors (Retryable)
- Connection timeouts
- Connection resets
- DNS failures
- Service temporarily unavailable

### Permanent Errors (Non-Retryable)
- Invalid chunk size
- Authorization failures
- File format errors
- Context cancellation (after checking)

## Performance Characteristics

### Upload
- **Chunking Overhead**: ~1-2% for metadata tracking
- **Memory**: O(1) per stream (no full file buffering)
- **Networking**: HTTP/2 multiplexing for parallel chunks
- **Retry Backoff**: Max 10 seconds between attempts (configurable)

### Download
- **Streaming**: Constant memory regardless of file size
- **Buffering**: 5MB chunks (same as upload)
- **Speed**: Network-limited (HTTP/2 H2C support)

## Future Enhancements

1. **Resumable Downloads**: Byte-range requests with checksum verification
2. **Adaptive Chunk Size**: Dynamic sizing based on network conditions
3. **Compression**: On-the-fly compression for bandwidth optimization
4. **Encryption**: End-to-end encryption for sensitive content
5. **Bandwidth Limiting**: Rate limiting for fair resource sharing
6. **Metrics**: Prometheus metrics for monitoring
7. **Circuit Breaker**: Fail fast on cascading failures

## Example: Client Implementation

```go
// Upload a 100MB file in chunks
file := os.Open("video.mp4")
defer file.Close()

// Initiate upload
initResp, _ := initUpload(videoID, int64(100*1024*1024))
uploadID := initResp.UploadID
chunkSize := initResp.ChunkSize

// Upload chunks
chunkIndex := 0
for {
    chunk := make([]byte, chunkSize)
    n, err := file.Read(chunk)
    if n == 0 || err == io.EOF {
        break
    }
    
    // Automatic retry on temporary errors
    uploadChunk(uploadID, chunkIndex, chunk[:n])
    chunkIndex++
    
    // Show progress
    progress, _ := getProgress(uploadID)
    fmt.Printf("Progress: %.1f%% (%.1f Mbps)\n", 
        progress.Percentage, progress.SpeedMbps)
}

// Complete upload
complete(uploadID)
```

## Debugging

### Enable Detailed Logging
```go
// In handlers, add structured logging:
log.Printf("Upload %s chunk %d: %d bytes (checksum: %s)",
    uploadID, chunkIndex, len(chunk), checksum)
```

### Check Progress Endpoints
```bash
# Monitor upload progress in real-time
watch -n 1 'curl -s http://localhost:8081/uploads/{id}/progress | jq'
```

### Verify Database State
```sql
SELECT * FROM uploads WHERE id = 'upload-id';
SELECT * FROM upload_chunks WHERE upload_id = 'upload-id';
```

### S3/MinIO Verification
```bash
# List uploaded chunks in S3
aws s3 ls s3://videos/video-id/chunks/

# Or with MinIO
mc ls minio/videos/video-id/chunks/
```
