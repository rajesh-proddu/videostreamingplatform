package bl

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
	"github.com/yourusername/videostreamingplatform/utils/events"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
)

// mockDB implements the dl.VideoStore interface used by VideoRepository.
// We need to go through VideoRepository since VideoService depends on it.
// For these tests we create a real VideoRepository backed by a simple
// in-memory VideoStore implementation.

type mockVideoStore struct {
	videos map[string]*models.Video
	nextID int
}

func newMockVideoStore() *mockVideoStore {
	return &mockVideoStore{videos: make(map[string]*models.Video)}
}

func (m *mockVideoStore) CreateVideo(ctx context.Context, video *models.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *mockVideoStore) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	v, ok := m.videos[id]
	if !ok {
		return nil, ErrInvalidVideoID
	}
	return v, nil
}

func (m *mockVideoStore) UpdateVideo(ctx context.Context, video *models.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *mockVideoStore) DeleteVideo(ctx context.Context, id string) error {
	delete(m.videos, id)
	return nil
}

func (m *mockVideoStore) ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error) {
	var result []*models.Video
	for _, v := range m.videos {
		result = append(result, v)
	}
	return result, nil
}

func setupServiceWithMockKafka(t *testing.T) (*VideoService, *kafka.MockProducer) {
	t.Helper()
	store := newMockVideoStore()
	repo := dl.NewVideoRepository(store)
	mock := kafka.NewMockProducer()
	svc := NewVideoService(repo, WithKafkaProducer(mock, nil))
	return svc, mock
}

func TestCreateVideo_PublishesEvent(t *testing.T) {
	svc, mock := setupServiceWithMockKafka(t)

	_, err := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Test Video",
		SizeBytes: 1024,
	})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	if len(mock.Messages) != 1 {
		t.Fatalf("expected 1 event, got %d", len(mock.Messages))
	}

	var evt events.VideoEvent
	if err := json.Unmarshal(mock.Messages[0].Value, &evt); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if evt.Type != events.VideoCreated {
		t.Errorf("expected event type %s, got %s", events.VideoCreated, evt.Type)
	}
	if evt.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", evt.Version)
	}
}

func TestUpdateVideo_PublishesEvent(t *testing.T) {
	svc, mock := setupServiceWithMockKafka(t)

	video, _ := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Test",
		SizeBytes: 100,
	})
	mock.Reset()

	_, err := svc.UpdateVideo(context.Background(), video.ID, &models.UpdateVideoRequest{
		Title: "Updated",
	})
	if err != nil {
		t.Fatalf("UpdateVideo failed: %v", err)
	}

	if len(mock.Messages) != 1 {
		t.Fatalf("expected 1 event, got %d", len(mock.Messages))
	}

	var evt events.VideoEvent
	_ = json.Unmarshal(mock.Messages[0].Value, &evt)
	if evt.Type != events.VideoUpdated {
		t.Errorf("expected %s, got %s", events.VideoUpdated, evt.Type)
	}
}

func TestDeleteVideo_PublishesEvent(t *testing.T) {
	svc, mock := setupServiceWithMockKafka(t)

	video, _ := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Test",
		SizeBytes: 100,
	})
	mock.Reset()

	if err := svc.DeleteVideo(context.Background(), video.ID); err != nil {
		t.Fatalf("DeleteVideo failed: %v", err)
	}

	if len(mock.Messages) != 1 {
		t.Fatalf("expected 1 event, got %d", len(mock.Messages))
	}

	var evt events.VideoEvent
	_ = json.Unmarshal(mock.Messages[0].Value, &evt)
	if evt.Type != events.VideoDeleted {
		t.Errorf("expected %s, got %s", events.VideoDeleted, evt.Type)
	}
}

func TestVideoService_NoProducer_NoError(t *testing.T) {
	store := newMockVideoStore()
	repo := dl.NewVideoRepository(store)
	svc := NewVideoService(repo) // no Kafka producer

	_, err := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "No Kafka",
		SizeBytes: 50,
	})
	if err != nil {
		t.Fatalf("CreateVideo should succeed without producer: %v", err)
	}
}
