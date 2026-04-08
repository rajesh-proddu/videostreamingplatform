package models

import "time"

type Video struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Duration       int       `json:"duration"` // seconds
	SizeBytes      int64     `json:"size_bytes"`
	Format         string    `json:"format"`          // video format (mp4, webm, etc)
	UploadProgress int       `json:"upload_progress"` // percentage 0-100
	UploadStatus   string    `json:"upload_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Upload struct {
	ID              string    `json:"id"`
	VideoID         string    `json:"video_id"`
	UserID          string    `json:"user_id"`
	ChunkCount      int       `json:"chunk_count"`
	CompletedChunks int       `json:"completed_chunks"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Download struct {
	ID          string     `json:"id"`
	VideoID     string     `json:"video_id"`
	UserID      string     `json:"user_id"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string     `json:"status"`
}

type CreateVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	SizeBytes   int64  `json:"size_bytes"`
	Format      string `json:"format"` // video format (mp4, webm, etc)
}

type UpdateVideoRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Format      string `json:"format,omitempty"`
}

type UploadChunkRequest struct {
	ChunkIndex  int    `json:"chunk_index"`
	ChunkData   []byte `json:"chunk_data"`
	TotalChunks int    `json:"total_chunks"`
}
