// Package metadata provides an HTTP client for the metadata service, used by
// dataservice to report upload lifecycle changes back to the record that owns
// the video. It mirrors utils/recommendations: construct with an empty baseURL
// to disable, and callers check Enabled() before use.
package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client communicates with the metadata service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a metadata client. Pass empty baseURL to disable.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Enabled returns true if the metadata service is configured.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

// MarkUploadComplete flips a video's upload_status to COMPLETED.
//
// A video record is created PENDING by metadataservice and nothing else moves
// it, so without this call every successfully uploaded video reads as pending
// forever. Callers treat failure as non-fatal: the bytes are already durable in
// S3, and a stale status is worth less than a failed upload.
func (c *Client) MarkUploadComplete(ctx context.Context, videoID string) error {
	return c.setUploadStatus(ctx, videoID, "COMPLETED")
}

// MarkUploadFailed flips a video's upload_status to FAILED.
func (c *Client) MarkUploadFailed(ctx context.Context, videoID string) error {
	return c.setUploadStatus(ctx, videoID, "FAILED")
}

func (c *Client) setUploadStatus(ctx context.Context, videoID, status string) error {
	if !c.Enabled() {
		return fmt.Errorf("metadata service not configured")
	}
	if videoID == "" {
		return fmt.Errorf("videoID is required")
	}

	body, err := json.Marshal(map[string]string{"upload_status": status})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/videos/%s", c.baseURL, videoID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("metadata service returned %d", resp.StatusCode)
	}
	return nil
}
