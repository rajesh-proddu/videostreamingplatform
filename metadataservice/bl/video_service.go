// Package bl provides business logic for the metadata service
package bl

import (
	"context"

	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

// VideoService handles business logic for video operations
type VideoService struct {
	repo *dl.VideoRepository
}

// NewVideoService creates a new video service
func NewVideoService(repo *dl.VideoRepository) *VideoService {
	return &VideoService{repo: repo}
}

// CreateVideo creates a new video with validation
func (s *VideoService) CreateVideo(ctx context.Context, req *models.CreateVideoRequest) (*models.Video, error) {
	// Validation
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}

	if req.SizeBytes == 0 {
		return nil, ErrInvalidSize
	}

	// Create via repository
	return s.repo.CreateVideo(ctx, req)
}

// GetVideo retrieves a video
func (s *VideoService) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	if id == "" {
		return nil, ErrInvalidVideoID
	}

	return s.repo.GetVideo(ctx, id)
}

// UpdateVideo updates a video
func (s *VideoService) UpdateVideo(ctx context.Context, id string, req *models.UpdateVideoRequest) (*models.Video, error) {
	if id == "" {
		return nil, ErrInvalidVideoID
	}

	return s.repo.UpdateVideo(ctx, id, req)
}

// DeleteVideo deletes a video
func (s *VideoService) DeleteVideo(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidVideoID
	}

	return s.repo.DeleteVideo(ctx, id)
}

// ListVideos lists all videos
func (s *VideoService) ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error) {
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	return s.repo.ListVideos(ctx, limit, offset)
}
