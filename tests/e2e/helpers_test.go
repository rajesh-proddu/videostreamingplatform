package e2e

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"
)

const (
	ChunkSize = 5 * 1024 * 1024 // 5 MB — matches server-side chunk size
)

// requireHealthy skips the test if either service is unreachable.
func requireHealthy(t *testing.T, c *Client) {
	t.Helper()
	if err := c.HealthMetadata(); err != nil {
		t.Skipf("metadata service unreachable: %v", err)
	}
	if err := c.HealthData(); err != nil {
		t.Skipf("data service unreachable: %v", err)
	}
}

// randomPayload returns n bytes of random data.
func randomPayload(t *testing.T, n int) []byte {
	t.Helper()
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return buf
}

// sha256sum returns the hex-encoded SHA-256 digest.
func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}

// chunkSlice splits data into chunks of the given size.
func chunkSlice(data []byte, size int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

// uploadFullVideo is a helper that creates a video, uploads it in chunks,
// and completes the upload. Returns the video ID and upload ID.
func uploadFullVideo(t *testing.T, c *Client, title string, payload []byte) (videoID, uploadID string) {
	t.Helper()

	// 1. Create video metadata
	video, err := c.CreateVideo(CreateVideoRequest{
		Title:       title,
		Description: "e2e test video",
		Duration:    60,
		SizeBytes:   int64(len(payload)),
		Format:      "mp4",
	})
	if err != nil {
		t.Fatalf("CreateVideo: %v", err)
	}
	videoID = video.ID
	t.Logf("Created video: %s", videoID)

	// 2. Initiate upload
	initResp, err := c.InitiateUpload(UploadInitiateRequest{
		VideoID:   videoID,
		UserID:    "e2e-agent",
		TotalSize: int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	uploadID = initResp.UploadID
	t.Logf("Upload initiated: %s", uploadID)

	// 3. Upload chunks
	chunks := chunkSlice(payload, ChunkSize)
	for i, chunk := range chunks {
		resp, err := c.UploadChunk(uploadID, i, chunk)
		if err != nil {
			t.Fatalf("UploadChunk[%d]: %v", i, err)
		}
		t.Logf("  Chunk %d/%d uploaded (status=%s)", i+1, len(chunks), resp.Status)
	}

	// 4. Complete upload
	complete, err := c.CompleteUpload(uploadID)
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	t.Logf("Upload completed: status=%s", complete.Status)

	return videoID, uploadID
}
