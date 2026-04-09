package e2e

import (
	"testing"
)

// TestDownloadAgent uploads a file, downloads it back, and verifies integrity
// via SHA-256 checksum comparison.
func TestDownloadAgent(t *testing.T) {
	cfg := LoadConfig(t)
	c := NewClient(cfg.MetadataURL, cfg.DataURL)
	requireHealthy(t, c)

	const fileSize = 6 * 1024 * 1024 // 6 MB → 2 chunks
	payload := randomPayload(t, fileSize)
	originalChecksum := sha256sum(payload)
	t.Logf("Original payload: %d bytes, sha256=%s", len(payload), originalChecksum)

	// Upload the video
	videoID, _ := uploadFullVideo(t, c, "Download Agent Test", payload)

	// Download it back
	t.Log("Downloading video...")
	downloaded, err := c.DownloadVideo(videoID)
	if err != nil {
		t.Fatalf("DownloadVideo: %v", err)
	}
	t.Logf("Downloaded: %d bytes", len(downloaded))

	// Verify size
	if len(downloaded) != len(payload) {
		t.Fatalf("size mismatch: uploaded %d bytes, downloaded %d bytes",
			len(payload), len(downloaded))
	}

	// Verify checksum
	downloadedChecksum := sha256sum(downloaded)
	t.Logf("Downloaded sha256=%s", downloadedChecksum)

	if originalChecksum != downloadedChecksum {
		t.Fatalf("checksum mismatch:\n  original:   %s\n  downloaded: %s",
			originalChecksum, downloadedChecksum)
	}

	t.Log("✓ Checksums match — download agent test passed")
}
