package dl

import (
	"context"
	"testing"

	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

func newTestUpload(id, videoID, userID string) *models.Upload {
	return &models.Upload{
		ID:      id,
		VideoID: videoID,
		UserID:  userID,
		Status:  "PENDING",
	}
}

func TestCreateUpload_Success(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	err := repo.CreateUpload(context.Background(), newTestUpload("u1", "v1", "user1"))
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}
}

func TestCreateUpload_NilUpload(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	err := repo.CreateUpload(context.Background(), nil)
	if err == nil {
		t.Error("CreateUpload(nil) should return error")
	}
}

func TestCreateUpload_Duplicate(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	upload := newTestUpload("u1", "v1", "user1")
	_ = repo.CreateUpload(context.Background(), upload)

	err := repo.CreateUpload(context.Background(), upload)
	if err == nil {
		t.Error("CreateUpload() with duplicate ID should return error")
	}
}

func TestGetUploadByID_Found(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	original := newTestUpload("u1", "v1", "user1")
	original.TotalSize = 5000
	_ = repo.CreateUpload(context.Background(), original)

	got, err := repo.GetUploadByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetUploadByID() error = %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("ID = %q, want %q", got.ID, "u1")
	}
	if got.VideoID != "v1" {
		t.Errorf("VideoID = %q, want %q", got.VideoID, "v1")
	}
	if got.TotalSize != 5000 {
		t.Errorf("TotalSize = %d, want 5000", got.TotalSize)
	}
}

func TestGetUploadByID_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	_, err := repo.GetUploadByID(context.Background(), "nonexistent")
	if err == nil {
		t.Error("GetUploadByID() should return error for non-existent ID")
	}
}

func TestListUploadsByUserID_Empty(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	uploads, err := repo.ListUploadsByUserID(context.Background(), "unknown-user")
	if err != nil {
		t.Fatalf("ListUploadsByUserID() error = %v", err)
	}
	if len(uploads) != 0 {
		t.Errorf("expected empty list, got %d", len(uploads))
	}
}

func TestListUploadsByUserID_FiltersCorrectly(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	_ = repo.CreateUpload(context.Background(), newTestUpload("u1", "v1", "alice"))
	_ = repo.CreateUpload(context.Background(), newTestUpload("u2", "v2", "alice"))
	_ = repo.CreateUpload(context.Background(), newTestUpload("u3", "v3", "bob"))

	aliceUploads, err := repo.ListUploadsByUserID(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListUploadsByUserID() error = %v", err)
	}
	if len(aliceUploads) != 2 {
		t.Errorf("alice uploads = %d, want 2", len(aliceUploads))
	}

	bobUploads, err := repo.ListUploadsByUserID(context.Background(), "bob")
	if err != nil {
		t.Fatalf("ListUploadsByUserID() error = %v", err)
	}
	if len(bobUploads) != 1 {
		t.Errorf("bob uploads = %d, want 1", len(bobUploads))
	}
}

func TestUpdateUpload_Success(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	upload := newTestUpload("u1", "v1", "user1")
	_ = repo.CreateUpload(context.Background(), upload)

	upload.Status = "COMPLETED"
	upload.UploadedSize = 5000
	err := repo.UpdateUpload(context.Background(), upload)
	if err != nil {
		t.Fatalf("UpdateUpload() error = %v", err)
	}

	got, _ := repo.GetUploadByID(context.Background(), "u1")
	if got.Status != "COMPLETED" {
		t.Errorf("Status = %q, want %q", got.Status, "COMPLETED")
	}
	if got.UploadedSize != 5000 {
		t.Errorf("UploadedSize = %d, want 5000", got.UploadedSize)
	}
}

func TestUpdateUpload_NilUpload(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	err := repo.UpdateUpload(context.Background(), nil)
	if err == nil {
		t.Error("UpdateUpload(nil) should return error")
	}
}

func TestUpdateUpload_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	err := repo.UpdateUpload(context.Background(), newTestUpload("nonexistent", "v1", "user1"))
	if err == nil {
		t.Error("UpdateUpload() should return error for non-existent ID")
	}
}

func TestDeleteUpload_Success(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	_ = repo.CreateUpload(context.Background(), newTestUpload("u1", "v1", "user1"))

	err := repo.DeleteUpload(context.Background(), "u1")
	if err != nil {
		t.Fatalf("DeleteUpload() error = %v", err)
	}

	_, err = repo.GetUploadByID(context.Background(), "u1")
	if err == nil {
		t.Error("upload should be gone after delete")
	}
}

func TestDeleteUpload_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryUploadRepository()
	err := repo.DeleteUpload(context.Background(), "nonexistent")
	if err == nil {
		t.Error("DeleteUpload() should return error for non-existent ID")
	}
}
