// Package bl provides business logic for the metadata service
package bl

import (
	"context"
	"log"
	"time"

	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
	"github.com/yourusername/videostreamingplatform/utils/cache"
	"github.com/yourusername/videostreamingplatform/utils/events"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
)

// VideoService handles business logic for video operations
type VideoService struct {
	repo               *dl.VideoRepository
	producer           kafka.Producer
	logger             *log.Logger
	cache              *cache.Cache
	cacheTTLGetVideo   time.Duration
	cacheTTLListVideos time.Duration
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

// WithCache configures the video service with Redis caching for reads.
func WithCache(c *cache.Cache, ttlGetVideo, ttlListVideos time.Duration) VideoServiceOption {
	return func(s *VideoService) {
		s.cache = c
		s.cacheTTLGetVideo = ttlGetVideo
		s.cacheTTLListVideos = ttlListVideos
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

	// Invalidate list cache (new video changes list results)
	s.invalidateListCache(ctx)

	s.publishEvent(ctx, events.VideoCreated, video)
	return video, nil
}

// GetVideo retrieves a video, with caching if enabled
func (s *VideoService) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	if id == "" {
		return nil, ErrInvalidVideoID
	}

	// Try cache first
	if s.cache != nil {
		var video models.Video
		if s.cache.Get(ctx, cache.VideoKey(id), &video) {
			return &video, nil
		}
	}

	// Cache miss or no cache—fetch from repository
	video, err := s.repo.GetVideo(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache for future reads
	if s.cache != nil && video != nil {
		s.cache.Set(ctx, cache.VideoKey(id), video, s.cacheTTLGetVideo)
	}

	return video, nil
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

	// Invalidate both individual and list caches
	s.invalidateVideoCache(ctx, id)

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

	// Invalidate both individual and list caches
	s.invalidateVideoCache(ctx, id)

	// CDN invalidation happens asynchronously via Kafka consumer (cdn-invalidator worker)
	s.publishEvent(ctx, events.VideoDeleted, map[string]string{"id": id})
	return nil
}

// ListVideos lists all videos, with caching if enabled
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

	// Try cache first
	if s.cache != nil {
		var videos []*models.Video
		if s.cache.Get(ctx, cache.ListKey(limit, offset), &videos) {
			return videos, nil
		}
	}

	videos, err := s.repo.ListVideos(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if s.cache != nil && len(videos) > 0 {
		s.cache.Set(ctx, cache.ListKey(limit, offset), videos, s.cacheTTLListVideos)
	}

	return videos, nil
}

// invalidateVideoCache removes both the individual video cache and all list caches.
func (s *VideoService) invalidateVideoCache(ctx context.Context, id string) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Delete(ctx, cache.VideoKey(id)); err != nil && s.logger != nil {
		s.logger.Printf("WARN: cache invalidation failed for key %s: %v", cache.VideoKey(id), err)
	}
	s.invalidateListCache(ctx)
}

// invalidateListCache removes all list cache entries.
func (s *VideoService) invalidateListCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	if err := s.cache.DeletePattern(ctx, "videos:list:*"); err != nil && s.logger != nil {
		s.logger.Printf("WARN: cache invalidation failed for pattern videos:list:*: %v", err)
	}
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
