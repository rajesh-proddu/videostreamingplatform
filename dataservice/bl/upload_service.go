// Package bl provides business logic operations for data management
package bl

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/videostreamingplatform/dataservice/dl"
	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

// UploadService handles upload-related business logic
type UploadService struct {
	uploadRepo dl.UploadRepository
	logger     *log.Logger
}

// NewUploadService creates a new upload service
func NewUploadService(uploadRepo dl.UploadRepository, logger *log.Logger) *UploadService {
	return &UploadService{
		uploadRepo: uploadRepo,
		logger:     logger,
	}
}

// InitiateUpload initiates a new upload session
func (s *UploadService) InitiateUpload(ctx context.Context, req *models.UploadInitiateRequest) (*models.Upload, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	uploadID := generateID("upl")
	now := time.Now()

	upload := &models.Upload{
		ID:             uploadID,
		VideoID:        req.VideoID,
		UserID:         req.UserID,
		TotalSize:      req.TotalSize,
		UploadedSize:   0,
		UploadedChunks: 0,
		TotalChunks:    calculateTotalChunks(req.TotalSize),
		Status:         "PENDING",
		Percentage:     0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.uploadRepo.CreateUpload(ctx, upload); err != nil {
		s.logger.Printf("Failed to create upload: %v", err)
		return nil, err
	}

	s.logger.Printf("Upload initiated: %s for user %s", uploadID, req.UserID)
	return upload, nil
}

// GetUploadProgress retrieves the current progress of an upload
func (s *UploadService) GetUploadProgress(ctx context.Context, uploadID string) (*models.Upload, error) {
	upload, err := s.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		s.logger.Printf("Failed to get upload progress: %v", err)
		return nil, err
	}

	return upload, nil
}

// RecordChunkUpload records the upload of a chunk
func (s *UploadService) RecordChunkUpload(ctx context.Context, uploadID string, chunkSize int64) (*models.Upload, error) {
	upload, err := s.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		s.logger.Printf("Failed to get upload: %v", err)
		return nil, err
	}

	// Update upload progress
	upload.UploadedSize += chunkSize
	upload.UploadedChunks++
	upload.Status = "IN_PROGRESS"
	upload.UpdatedAt = time.Now()

	// Calculate percentage
	if upload.TotalSize > 0 {
		upload.Percentage = (float64(upload.UploadedSize) / float64(upload.TotalSize)) * 100
	}

	// Calculate speed and estimated time
	if upload.Percentage > 0 {
		elapsed := time.Since(upload.CreatedAt).Seconds()
		if elapsed > 0 {
			uploadedMB := float64(upload.UploadedSize) / (1024 * 1024)
			upload.SpeedMbps = uploadedMB / elapsed
			remainingBytes := upload.TotalSize - upload.UploadedSize
			if upload.SpeedMbps > 0 {
				remainingSeconds := int64((float64(remainingBytes) / (1024 * 1024)) / upload.SpeedMbps)
				upload.EstimatedSeconds = &remainingSeconds
			}
		}
	}

	if err := s.uploadRepo.UpdateUpload(ctx, upload); err != nil {
		s.logger.Printf("Failed to update upload progress: %v", err)
		return nil, err
	}

	return upload, nil
}

// CompleteUpload marks an upload as completed
func (s *UploadService) CompleteUpload(ctx context.Context, uploadID string) (*models.Upload, error) {
	upload, err := s.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		s.logger.Printf("Failed to get upload: %v", err)
		return nil, err
	}

	now := time.Now()
	upload.Status = "COMPLETED"
	upload.Percentage = 100
	upload.UpdatedAt = now
	upload.CompletedAt = &now

	if err := s.uploadRepo.UpdateUpload(ctx, upload); err != nil {
		s.logger.Printf("Failed to complete upload: %v", err)
		return nil, err
	}

	s.logger.Printf("Upload completed: %s", uploadID)
	return upload, nil
}

// FailUpload marks an upload as failed
func (s *UploadService) FailUpload(ctx context.Context, uploadID string) (*models.Upload, error) {
	upload, err := s.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		s.logger.Printf("Failed to get upload: %v", err)
		return nil, err
	}

	now := time.Now()
	upload.Status = "FAILED"
	upload.UpdatedAt = now
	upload.CompletedAt = &now

	if err := s.uploadRepo.UpdateUpload(ctx, upload); err != nil {
		s.logger.Printf("Failed to mark upload as failed: %v", err)
		return nil, err
	}

	s.logger.Printf("Upload failed: %s", uploadID)
	return upload, nil
}

// ListUserUploads lists all uploads for a user
func (s *UploadService) ListUserUploads(ctx context.Context, userID string) ([]*models.Upload, error) {
	uploads, err := s.uploadRepo.ListUploadsByUserID(ctx, userID)
	if err != nil {
		s.logger.Printf("Failed to list user uploads: %v", err)
		return nil, err
	}

	return uploads, nil
}

// Helper functions

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func calculateTotalChunks(totalSize int64) int {
	const chunkSize = 5 * 1024 * 1024 // 5 MB chunks
	chunks := totalSize / chunkSize
	if totalSize%chunkSize > 0 {
		chunks++
	}
	return int(chunks)
}
