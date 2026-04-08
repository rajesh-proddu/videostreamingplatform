// Package streaming provides utilities for handling streaming uploads
package streaming

import (
	"crypto/md5"
	"io"
)

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

func CalculateMD5(data io.ReadCloser) (string, error) {
	defer data.Close()
	h := md5.New()
	if _, err := io.Copy(h, data); err != nil {
		return "", err
	}
	return string(h.Sum(nil)), nil
}

func (s *UploadSession) MarkChunkReceived(index int, checksum string) {
	s.ReceivedChunks[index] = true
}

func (s *UploadSession) IsComplete() bool {
	totalChunks := s.TotalSize / s.ChunkSize
	if s.TotalSize%s.ChunkSize > 0 {
		totalChunks++
	}
	return int64(len(s.ReceivedChunks)) == totalChunks
}

func (s *UploadSession) GetMissingChunks() []int {
	totalChunks := int(s.TotalSize / s.ChunkSize)
	if s.TotalSize%s.ChunkSize > 0 {
		totalChunks++
	}
	var missing []int
	for i := 0; i < totalChunks; i++ {
		if !s.ReceivedChunks[i] {
			missing = append(missing, i)
		}
	}
	return missing
}
