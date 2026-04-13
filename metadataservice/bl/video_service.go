// Package bl provides business logic for the metadata service
package bl

import (
	"context"
	"log"

	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
	"github.com/yourusername/videostreamingplatform/utils/events"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
)

// VideoService handles business logic for video operations
type VideoService struct {
	repo     *dl.VideoRepository
	producer kafka.Producer
	logger   *log.Logger
}

// NewVideoService creates a new video service
func NewVideoService(repo *dl.VideoRepository, opts ...VideoServiceOption) *VideoService {
	s := &VideoService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// VideoServiceOption configures a VideoService.
type VideoServiceOption func(*VideoService)

// WithKafkaProducer configures the video service with a Kafka producer for lifecycle events.
func WithKafkaProducer(p kafka.Producer, logger *log.Logger) VideoServiceOption {
	return func(s *VideoService) {
		s.producer = p
		s.logger = logger
	}
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
	video, err := s.repo.CreateVideo(ctx, req)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, events.VideoCreated, video)
	return video, nil
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

	video, err := s.repo.UpdateVideo(ctx, id, req)
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, events.VideoUpdated, video)
	return video, nil
}

// DeleteVideo deletes a video
func (s *VideoService) DeleteVideo(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidVideoID
	}

	if err := s.repo.DeleteVideo(ctx, id); err != nil {
		return err
	}

	s.publishEvent(ctx, events.VideoDeleted, map[string]string{"id": id})
	return nil
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

// publishEvent publishes an event to Kafka (best-effort: logs errors, never fails the API call).
func (s *VideoService) publishEvent(ctx context.Context, eventType string, payload any) {
	if s.producer == nil {
		return
	}

	evt := events.NewVideoEvent(eventType, payload)
	data, err := evt.Marshal()
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("Failed to marshal %s event: %v", eventType, err)
		}
		return
	}

	if err := s.producer.Publish(ctx, []byte(eventType), data); err != nil {
		if s.logger != nil {
			s.logger.Printf("Failed to publish %s event: %v", eventType, err)
		}
	}
}
