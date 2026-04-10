package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ---------- Metadata Service types ----------

type Video struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Duration       int    `json:"duration"`
	SizeBytes      int64  `json:"size_bytes"`
	Format         string `json:"format"`
	UploadProgress int    `json:"upload_progress"`
	UploadStatus   string `json:"upload_status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type CreateVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	SizeBytes   int64  `json:"size_bytes"`
	Format      string `json:"format"`
}

type UpdateVideoRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Format      string `json:"format,omitempty"`
}

type ListVideosResponse struct {
	Videos []Video `json:"videos"`
	Count  int     `json:"count"`
}

// ---------- Data Service types ----------

type UploadInitiateRequest struct {
	VideoID   string `json:"video_id"`
	UserID    string `json:"user_id"`
	TotalSize int64  `json:"total_size"`
}

type UploadInitiateResponse struct {
	UploadID  string `json:"upload_id"`
	ChunkSize int64  `json:"chunk_size"`
	Message   string `json:"message"`
}

type UploadChunkResponse struct {
	Status  string `json:"status"`
	VideoID string `json:"video_id"`
}

type UploadProgressResponse struct {
	UploadID         string  `json:"upload_id"`
	Percentage       float64 `json:"percentage"`
	UploadedBytes    int64   `json:"uploaded_bytes"`
	TotalBytes       int64   `json:"total_bytes"`
	UploadedChunks   int     `json:"uploaded_chunks"`
	TotalChunks      int     `json:"total_chunks"`
	SpeedMbps        float64 `json:"speed_mbps"`
	EstimatedSeconds *int64  `json:"estimated_seconds,omitempty"`
}

type CompleteUploadResponse struct {
	UploadID string `json:"upload_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ---------- Client ----------

// Client wraps both service APIs.
type Client struct {
	MetadataURL string
	DataURL     string
	HTTP        *http.Client
}

// NewClient creates an HTTP client pointed at the given service URLs.
func NewClient(metadataURL, dataURL string) *Client {
	return &Client{
		MetadataURL: metadataURL,
		DataURL:     dataURL,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ---------- Health ----------

func (c *Client) HealthMetadata() error {
	resp, err := c.HTTP.Get(c.MetadataURL + "/health")
	if err != nil {
		return fmt.Errorf("metadata health: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metadata health: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) HealthData() error {
	resp, err := c.HTTP.Get(c.DataURL + "/health")
	if err != nil {
		return fmt.Errorf("data health: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("data health: status %d", resp.StatusCode)
	}
	return nil
}

// ---------- Video CRUD ----------

func (c *Client) CreateVideo(req CreateVideoRequest) (*Video, error) {
	body, _ := json.Marshal(req)
	resp, err := c.HTTP.Post(c.MetadataURL+"/videos", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create video: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create video: status %d: %s", resp.StatusCode, string(b))
	}
	var v Video
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("create video decode: %w", err)
	}
	return &v, nil
}

func (c *Client) GetVideo(id string) (*Video, error) {
	resp, err := c.HTTP.Get(c.MetadataURL + "/videos/" + id)
	if err != nil {
		return nil, fmt.Errorf("get video: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get video: status %d: %s", resp.StatusCode, string(b))
	}
	var v Video
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("get video decode: %w", err)
	}
	return &v, nil
}

func (c *Client) UpdateVideo(id string, req UpdateVideoRequest) (*Video, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPut, c.MetadataURL+"/videos/"+id, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("update video: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update video: status %d: %s", resp.StatusCode, string(b))
	}
	var v Video
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("update video decode: %w", err)
	}
	return &v, nil
}

func (c *Client) DeleteVideo(id string) error {
	httpReq, _ := http.NewRequest(http.MethodDelete, c.MetadataURL+"/videos/"+id, nil)
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("delete video: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete video: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) ListVideos(limit, offset int) (*ListVideosResponse, error) {
	url := fmt.Sprintf("%s/videos?limit=%d&offset=%d", c.MetadataURL, limit, offset)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("list videos: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list videos: status %d: %s", resp.StatusCode, string(b))
	}
	var lr ListVideosResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("list videos decode: %w", err)
	}
	return &lr, nil
}

// ---------- Upload ----------

func (c *Client) InitiateUpload(req UploadInitiateRequest) (*UploadInitiateResponse, error) {
	body, _ := json.Marshal(req)
	resp, err := c.HTTP.Post(c.DataURL+"/uploads/initiate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("initiate upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("initiate upload: status %d: %s", resp.StatusCode, string(b))
	}
	var ir UploadInitiateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return nil, fmt.Errorf("initiate upload decode: %w", err)
	}
	return &ir, nil
}

func (c *Client) UploadChunk(uploadID string, chunkIndex int, data []byte) (*UploadChunkResponse, error) {
	url := fmt.Sprintf("%s/uploads/%s/chunks?chunkIndex=%d", c.DataURL, uploadID, chunkIndex)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload chunk %d: %w", chunkIndex, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload chunk %d: status %d: %s", chunkIndex, resp.StatusCode, string(b))
	}
	var cr UploadChunkResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("upload chunk decode: %w", err)
	}
	return &cr, nil
}

func (c *Client) GetUploadProgress(uploadID string) (*UploadProgressResponse, error) {
	resp, err := c.HTTP.Get(fmt.Sprintf("%s/uploads/%s/progress", c.DataURL, uploadID))
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get progress: status %d: %s", resp.StatusCode, string(b))
	}
	var pr UploadProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("get progress decode: %w", err)
	}
	return &pr, nil
}

func (c *Client) CompleteUpload(uploadID string) (*CompleteUploadResponse, error) {
	resp, err := c.HTTP.Post(fmt.Sprintf("%s/uploads/%s/complete", c.DataURL, uploadID), "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("complete upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("complete upload: status %d: %s", resp.StatusCode, string(b))
	}
	var cr CompleteUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("complete upload decode: %w", err)
	}
	return &cr, nil
}

// ---------- Download ----------

func (c *Client) DownloadVideo(videoID string) ([]byte, error) {
	resp, err := c.HTTP.Get(fmt.Sprintf("%s/videos/%s/download", c.DataURL, videoID))
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download: status %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}
