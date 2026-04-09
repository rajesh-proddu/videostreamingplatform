package e2e

import (
	"fmt"
	"sync"
	"testing"
)

// TestConcurrentAgents simulates multiple upload/download agents working
// in parallel to verify the platform handles concurrent transfers.
func TestConcurrentAgents(t *testing.T) {
	cfg := LoadConfig(t)
	c := NewClient(cfg.MetadataURL, cfg.DataURL)
	requireHealthy(t, c)

	const (
		numAgents = 5
		fileSize  = 3 * 1024 * 1024 // 3 MB per agent
	)

	var wg sync.WaitGroup
	errors := make(chan error, numAgents)

	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(agentNum int) {
			defer wg.Done()

			title := fmt.Sprintf("Concurrent Agent %d", agentNum)
			payload := make([]byte, fileSize)
			// Fill with deterministic but unique data per agent
			for j := range payload {
				payload[j] = byte((agentNum*251 + j*37) % 256)
			}
			originalChecksum := sha256sum(payload)

			// Upload
			videoID, _ := uploadFullVideo(t, c, title, payload)

			// Download and verify
			downloaded, err := c.DownloadVideo(videoID)
			if err != nil {
				errors <- fmt.Errorf("agent %d download: %w", agentNum, err)
				return
			}

			if sha256sum(downloaded) != originalChecksum {
				errors <- fmt.Errorf("agent %d: checksum mismatch", agentNum)
				return
			}

			t.Logf("Agent %d: ✓ upload + download verified (video=%s)", agentNum, videoID)
		}(i)
	}

	wg.Wait()
	close(errors)

	var failed int
	for err := range errors {
		t.Error(err)
		failed++
	}

	if failed == 0 {
		t.Logf("✓ All %d concurrent agents completed successfully", numAgents)
	}
}
