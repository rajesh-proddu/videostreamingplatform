package bl

import (
	"context"
	"testing"

	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

// A video is created PENDING and only dataservice knows when the object is
// actually complete, so PUT /videos/{id} has to be able to carry the status.
// Without this the catalog reports every uploaded video as pending forever.
func TestUpdateVideo_SetsUploadStatus(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)
	ctx := context.Background()

	created, err := svc.CreateVideo(ctx, &models.CreateVideoRequest{
		Title:     "Freshly uploaded",
		SizeBytes: 4096,
	})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}
	if created.UploadStatus != "PENDING" {
		t.Fatalf("new video upload_status = %q, want PENDING", created.UploadStatus)
	}

	updated, err := svc.UpdateVideo(ctx, created.ID, &models.UpdateVideoRequest{
		UploadStatus: "COMPLETED",
	})
	if err != nil {
		t.Fatalf("UpdateVideo failed: %v", err)
	}
	if updated.UploadStatus != "COMPLETED" {
		t.Errorf("upload_status = %q, want COMPLETED", updated.UploadStatus)
	}

	// A status-only update must not blank out the rest of the record.
	if updated.Title != "Freshly uploaded" {
		t.Errorf("title = %q, want it preserved", updated.Title)
	}
	if updated.SizeBytes != 4096 {
		t.Errorf("size_bytes = %d, want it preserved", updated.SizeBytes)
	}

	got, err := svc.GetVideo(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetVideo failed: %v", err)
	}
	if got.UploadStatus != "COMPLETED" {
		t.Errorf("persisted upload_status = %q, want COMPLETED", got.UploadStatus)
	}
}

// An update that says nothing about the status must leave it alone.
func TestUpdateVideo_OmittedUploadStatusIsPreserved(t *testing.T) {
	t.Parallel()

	svc := setupServiceNoCache(t)
	ctx := context.Background()

	created, err := svc.CreateVideo(ctx, &models.CreateVideoRequest{Title: "Keep status", SizeBytes: 10})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}
	if _, err := svc.UpdateVideo(ctx, created.ID, &models.UpdateVideoRequest{UploadStatus: "COMPLETED"}); err != nil {
		t.Fatalf("UpdateVideo failed: %v", err)
	}

	renamed, err := svc.UpdateVideo(ctx, created.ID, &models.UpdateVideoRequest{Title: "Renamed"})
	if err != nil {
		t.Fatalf("UpdateVideo failed: %v", err)
	}
	if renamed.UploadStatus != "COMPLETED" {
		t.Errorf("upload_status = %q, want it left at COMPLETED", renamed.UploadStatus)
	}
	if renamed.Title != "Renamed" {
		t.Errorf("title = %q, want Renamed", renamed.Title)
	}
}
