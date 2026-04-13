// Package streaming provides utilities for handling streaming uploads
package streaming

const DefaultChunkSize = 5 * 1024 * 1024

type UploadSession struct {
	UploadID       string
	VideoID        string
	TotalSize      int64
	ChunkSize      int64
	ReceivedChunks map[int]bool
}

func NewUploadSession(id, videoID string, size, chunkSize int64) *UploadSession {
	return &UploadSession{
		UploadID:       id,
		VideoID:        videoID,
		TotalSize:      size,
		ChunkSize:      chunkSize,
		ReceivedChunks: make(map[int]bool),
	}
}
