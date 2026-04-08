// Package models defines data models for the data service
package models

import "time"

// Upload represents an upload session
type Upload struct {
	ID               string     `json:"id"`
	VideoID          string     `json:"video_id"`
	UserID           string     `json:"user_id"`
	TotalSize        int64      `json:"total_size"`
	UploadedSize     int64      `json:"uploaded_size"`
	UploadedChunks   int        `json:"uploaded_chunks"`
	TotalChunks      int        `json:"total_chunks"`
	Status           string     `json:"status"` // PENDING, IN_PROGRESS, COMPLETED, FAILED
	Percentage       float64    `json:"percentage"`
	SpeedMbps        float64    `json:"speed_mbps,omitempty"`
	EstimatedSeconds *int64     `json:"estimated_seconds,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// UploadInitiateRequest represents a request to initiate an upload
type UploadInitiateRequest struct {
	VideoID   string `json:"video_id"`
	UserID    string `json:"user_id"`
	TotalSize int64  `json:"total_size"`
}

// UploadInitiateResponse represents the response from initiating an upload
type UploadInitiateResponse struct {
	UploadID  string `json:"upload_id"`
	ChunkSize int64  `json:"chunk_size"`
	Message   string `json:"message"`
}

// UploadChunkRequest represents a chunk upload request
type UploadChunkRequest struct {
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	Checksum    string `json:"checksum"`
}

// UploadProgressResponse represents upload progress
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

// CompleteUploadRequest represents a request to complete an upload
type CompleteUploadRequest struct {
	FinalChecksum string `json:"final_checksum"`
}

// CompleteUploadResponse represents the response from completing an upload
type CompleteUploadResponse struct {
	UploadID string `json:"upload_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}
