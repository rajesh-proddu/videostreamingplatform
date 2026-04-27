// Package dl provides data layer operations for the metadata service
package dl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

// ErrVideoNotFound is returned when a video lookup yields no row.
var ErrVideoNotFound = errors.New("video not found")

// VideoStore defines the database operations required by VideoRepository.
type VideoStore interface {
	CreateVideo(ctx context.Context, video *models.Video) error
	GetVideo(ctx context.Context, id string) (*models.Video, error)
	UpdateVideo(ctx context.Context, video *models.Video) error
	DeleteVideo(ctx context.Context, id string) error
	ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error)
}

// VideoRepository handles video data operations
type VideoRepository struct {
	db VideoStore
}

// NewVideoRepository creates a new video repository
func NewVideoRepository(database VideoStore) *VideoRepository {
	return &VideoRepository{db: database}
}

// CreateVideo creates a new video record
func (r *VideoRepository) CreateVideo(ctx context.Context, req *models.CreateVideoRequest) (*models.Video, error) {
	dbVideo := &models.Video{
		ID:           uuid.New().String(),
		Title:        req.Title,
		Description:  req.Description,
		Duration:     req.Duration,
		SizeBytes:    req.SizeBytes,
		UploadStatus: "PENDING",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := r.db.CreateVideo(ctx, dbVideo); err != nil {
		return nil, fmt.Errorf("failed to create video: %w", err)
	}

	return &models.Video{
		ID:             dbVideo.ID,
		Title:          dbVideo.Title,
		Description:    dbVideo.Description,
		Duration:       dbVideo.Duration,
		SizeBytes:      dbVideo.SizeBytes,
		Format:         req.Format,
		UploadStatus:   dbVideo.UploadStatus,
		CreatedAt:      dbVideo.CreatedAt,
		UpdatedAt:      dbVideo.UpdatedAt,
		UploadProgress: 0,
	}, nil
}

// GetVideo retrieves a video by ID
func (r *VideoRepository) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	dbVideo, err := r.db.GetVideo(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	if dbVideo == nil {
		return nil, fmt.Errorf("%w: %s", ErrVideoNotFound, id)
	}

	return &models.Video{
		ID:           dbVideo.ID,
		Title:        dbVideo.Title,
		Description:  dbVideo.Description,
		Duration:     dbVideo.Duration,
		SizeBytes:    dbVideo.SizeBytes,
		UploadStatus: dbVideo.UploadStatus,
		CreatedAt:    dbVideo.CreatedAt,
		UpdatedAt:    dbVideo.UpdatedAt,
	}, nil
}

// UpdateVideo updates a video's metadata
func (r *VideoRepository) UpdateVideo(ctx context.Context, id string, req *models.UpdateVideoRequest) (*models.Video, error) {
	video, err := r.GetVideo(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		video.Title = req.Title
	}
	if req.Description != "" {
		video.Description = req.Description
	}
	if req.Duration > 0 {
		video.Duration = req.Duration
	}
	if req.SizeBytes > 0 {
		video.SizeBytes = req.SizeBytes
	}
	if req.Format != "" {
		video.Format = req.Format
	}

	video.UpdatedAt = time.Now()

	// Convert back to db.Video for storage
	dbVideo := &models.Video{
		ID:           video.ID,
		Title:        video.Title,
		Description:  video.Description,
		Duration:     video.Duration,
		SizeBytes:    video.SizeBytes,
		UploadStatus: video.UploadStatus,
		CreatedAt:    video.CreatedAt,
		UpdatedAt:    video.UpdatedAt,
	}

	if err := r.db.UpdateVideo(ctx, dbVideo); err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	return video, nil
}

// DeleteVideo deletes a video
func (r *VideoRepository) DeleteVideo(ctx context.Context, id string) error {
	if err := r.db.DeleteVideo(ctx, id); err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}
	return nil
}

// ListVideos lists all videos
func (r *VideoRepository) ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error) {
	dbVideos, err := r.db.ListVideos(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}

	videos := make([]*models.Video, len(dbVideos))
	for i, dbVideo := range dbVideos {
		videos[i] = &models.Video{
			ID:           dbVideo.ID,
			Title:        dbVideo.Title,
			Description:  dbVideo.Description,
			Duration:     dbVideo.Duration,
			SizeBytes:    dbVideo.SizeBytes,
			UploadStatus: dbVideo.UploadStatus,
			CreatedAt:    dbVideo.CreatedAt,
			UpdatedAt:    dbVideo.UpdatedAt,
		}
	}

	return videos, nil
}
