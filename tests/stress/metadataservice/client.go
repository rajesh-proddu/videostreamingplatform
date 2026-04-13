package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type video struct {
	ID string `json:"id"`
}

type createVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	SizeBytes   int64  `json:"size_bytes"`
	Format      string `json:"format"`
}

type updateVideoRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Format      string `json:"format,omitempty"`
}

type metadataClient struct {
	baseURL string
	http    *http.Client
}

func newMetadataClient(cfg config) *metadataClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = cfg.Workers * 4
	transport.MaxIdleConnsPerHost = cfg.Workers * 4
	transport.MaxConnsPerHost = cfg.Workers * 4

	return &metadataClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: transport,
		},
	}
}

func (c *metadataClient) health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("health check failed: status=%d body=%q", resp.StatusCode, string(body))
	}

	return nil
}

func (c *metadataClient) createVideo(ctx context.Context, runTag string, seq uint64) (string, int, error) {
	payload := createVideoRequest{
		Title:       fmt.Sprintf("stress-%s-title-%d", runTag, seq),
		Description: fmt.Sprintf("stress-%s-description-%d", runTag, seq),
		Duration:    60 + int(seq%3600),
		SizeBytes:   int64(1024 + (seq % 10_000)),
		Format:      "mp4",
	}

	var created video
	status, err := c.doJSON(ctx, http.MethodPost, "/videos", payload, &created)
	if err != nil {
		return "", status, err
	}

	return created.ID, status, nil
}

func (c *metadataClient) getVideo(ctx context.Context, id string) (int, error) {
	return c.doJSON(ctx, http.MethodGet, "/videos/"+id, nil, nil)
}

func (c *metadataClient) listVideos(ctx context.Context, limit, offset int) (int, error) {
	return c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/videos?limit=%d&offset=%d", limit, offset), nil, nil)
}

func (c *metadataClient) updateVideo(ctx context.Context, id, runTag string, seq uint64) (int, error) {
	payload := updateVideoRequest{
		Title:       fmt.Sprintf("stress-%s-updated-%d", runTag, seq),
		Description: fmt.Sprintf("stress-%s-updated-description-%d", runTag, seq),
		Duration:    120 + int(seq%7200),
		SizeBytes:   int64(2048 + (seq % 20_000)),
		Format:      "webm",
	}

	return c.doJSON(ctx, http.MethodPut, "/videos/"+id, payload, nil)
}

func (c *metadataClient) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) (int, error) {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		_ = start
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("status=%d body=%q", resp.StatusCode, string(body))
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, nil
}
