package e2e

import (
	"testing"
)

// TestUploadAgent exercises the full upload lifecycle:
//
//	health → create video → initiate upload → send chunks → track progress → complete
func TestUploadAgent(t *testing.T) {
	cfg := LoadConfig(t)
	c := NewClient(cfg.MetadataURL, cfg.DataURL)
	requireHealthy(t, c)

	const fileSize = 12 * 1024 * 1024 // 12 MB → 3 chunks of 5 MB + 1 of 2 MB
	payload := randomPayload(t, fileSize)

	// 1. Create video metadata
	video, err := c.CreateVideo(CreateVideoRequest{
		Title:       "Upload Agent Test",
		Description: "e2e upload lifecycle test",
		Duration:    120,
		SizeBytes:   int64(fileSize),
		Format:      "mp4",
	})
	if err != nil {
		t.Fatalf("CreateVideo: %v", err)
	}
	if video.ID == "" {
		t.Fatal("CreateVideo returned empty ID")
	}
	t.Logf("Video created: id=%s title=%q", video.ID, video.Title)

	// 2. Initiate upload session
	initResp, err := c.InitiateUpload(UploadInitiateRequest{
		VideoID:   video.ID,
		UserID:    "e2e-upload-agent",
		TotalSize: int64(fileSize),
	})
	if err != nil {
		t.Fatalf("InitiateUpload: %v", err)
	}
	if initResp.UploadID == "" {
		t.Fatal("InitiateUpload returned empty upload_id")
	}
	t.Logf("Upload initiated: upload_id=%s", initResp.UploadID)

	// 3. Upload chunks
	chunks := chunkSlice(payload, ChunkSize)
	t.Logf("Uploading %d chunks (%d bytes each, last=%d bytes)",
		len(chunks), ChunkSize, len(chunks[len(chunks)-1]))

	for i, chunk := range chunks {
		resp, err := c.UploadChunk(initResp.UploadID, i, chunk)
		if err != nil {
			t.Fatalf("UploadChunk[%d/%d]: %v", i+1, len(chunks), err)
		}
		t.Logf("  Chunk %d/%d → status=%s", i+1, len(chunks), resp.Status)

		// Check progress after each chunk
		progress, err := c.GetUploadProgress(initResp.UploadID)
		if err != nil {
			t.Fatalf("GetUploadProgress after chunk %d: %v", i+1, err)
		}
		t.Logf("  Progress: %.1f%% (%d/%d chunks, %.2f Mbps)",
			progress.Percentage, progress.UploadedChunks, progress.TotalChunks, progress.SpeedMbps)

		if progress.UploadedChunks != i+1 {
			t.Errorf("expected %d uploaded chunks, got %d", i+1, progress.UploadedChunks)
		}
	}

	// 4. Complete upload
	complete, err := c.CompleteUpload(initResp.UploadID)
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	t.Logf("Upload completed: status=%s", complete.Status)

	if complete.Status != "completed" && complete.Status != "COMPLETED" {
		t.Errorf("expected status completed, got %q", complete.Status)
	}

	// 5. Final progress should be 100%
	finalProgress, err := c.GetUploadProgress(initResp.UploadID)
	if err != nil {
		t.Fatalf("Final GetUploadProgress: %v", err)
	}
	if finalProgress.Percentage < 99.9 {
		t.Errorf("expected ~100%% progress, got %.1f%%", finalProgress.Percentage)
	}
	t.Logf("Final progress: %.1f%% — upload agent test passed", finalProgress.Percentage)
}
