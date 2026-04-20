package bl

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

func setupServiceNoCache(t *testing.T) *VideoService {
	t.Helper()
	store := newMockVideoStore()
	repo := dl.NewVideoRepository(store)
	return NewVideoService(repo) // no cache, no Kafka
}

func setupServiceWithCacheTTL(t *testing.T) *VideoService {
	t.Helper()
	store := newMockVideoStore()
	repo := dl.NewVideoRepository(store)
	// Use nil cache — tests verify nil-safe path doesn't crash
	// Integration with real Redis is tested in CI with Redis service
	return NewVideoService(repo, WithCache(nil, 5*time.Minute, 1*time.Minute))
}

func TestGetVideo_WithoutCache_Works(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)
	created, err := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "No Cache Video",
		SizeBytes: 1024,
	})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	got, err := svc.GetVideo(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetVideo failed: %v", err)
	}
	if got.Title != "No Cache Video" {
		t.Errorf("title = %q, want %q", got.Title, "No Cache Video")
	}
}

func TestGetVideo_WithNilCache_DoesNotPanic(t *testing.T) {
	t.Parallel()

	svc := setupServiceWithCacheTTL(t)
	created, err := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Nil Cache Video",
		SizeBytes: 2048,
	})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	got, err := svc.GetVideo(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetVideo failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestListVideos_DefaultLimit(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)
	for i := 0; i < 3; i++ {
		_, _ = svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
			Title:     "Video",
			SizeBytes: 100,
		})
	}

	videos, err := svc.ListVideos(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListVideos failed: %v", err)
	}
	if len(videos) != 3 {
		t.Errorf("got %d videos, want 3", len(videos))
	}
}

func TestListVideos_MaxLimit(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)
	// Request limit > 100 should be clamped to 100
	videos, err := svc.ListVideos(context.Background(), 200, 0)
	if err != nil {
		t.Fatalf("ListVideos failed: %v", err)
	}
	_ = videos // just verify no panic
}

func TestUpdateVideo_InvalidatesCache_NilSafe(t *testing.T) {
	t.Parallel()

	svc := setupServiceWithCacheTTL(t)
	created, _ := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Original",
		SizeBytes: 512,
	})

	updated, err := svc.UpdateVideo(context.Background(), created.ID, &models.UpdateVideoRequest{
		Title: "Updated",
	})
	if err != nil {
		t.Fatalf("UpdateVideo failed: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("title = %q, want %q", updated.Title, "Updated")
	}
}

func TestDeleteVideo_InvalidatesCache_NilSafe(t *testing.T) {
	t.Parallel()

	svc := setupServiceWithCacheTTL(t)
	created, _ := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "To Delete",
		SizeBytes: 256,
	})

	if err := svc.DeleteVideo(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteVideo failed: %v", err)
	}

	// Verify it's actually deleted
	_, err := svc.GetVideo(context.Background(), created.ID)
	if err == nil {
		t.Error("GetVideo after delete should return error")
	}
}

func TestCreateVideo_Validation(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)

	_, err := svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "",
		SizeBytes: 100,
	})
	if err != ErrInvalidTitle {
		t.Errorf("empty title: err = %v, want ErrInvalidTitle", err)
	}

	_, err = svc.CreateVideo(context.Background(), &models.CreateVideoRequest{
		Title:     "Valid",
		SizeBytes: 0,
	})
	if err != ErrInvalidSize {
		t.Errorf("zero size: err = %v, want ErrInvalidSize", err)
	}
}

func TestGetVideo_EmptyID_ReturnsError(t *testing.T) {
	t.Parallel()
	svc := setupServiceNoCache(t)
	_, err := svc.GetVideo(context.Background(), "")
	if err != ErrInvalidVideoID {
		t.Errorf("err = %v, want ErrInvalidVideoID", err)
	}
}

func TestUpdateVideo_EmptyID_ReturnsError(t *testing.T) {
	t.Parallel()
	svc := setupServiceNoCache(t)
	_, err := svc.UpdateVideo(context.Background(), "", &models.UpdateVideoRequest{Title: "x"})
	if err != ErrInvalidVideoID {
		t.Errorf("err = %v, want ErrInvalidVideoID", err)
	}
}

func TestDeleteVideo_EmptyID_ReturnsError(t *testing.T) {
	t.Parallel()
	svc := setupServiceNoCache(t)
	err := svc.DeleteVideo(context.Background(), "")
	if err != ErrInvalidVideoID {
		t.Errorf("err = %v, want ErrInvalidVideoID", err)
	}
}
