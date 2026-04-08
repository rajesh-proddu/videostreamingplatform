// Package bl_test provides tests for the upload service
package bl

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/yourusername/videostreamingplatform/dataservice/dl"
	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

func TestInitiateUpload(t *testing.T) {
	logger := log.New(os.Stdout, "[Test] ", log.LstdFlags)
	uploadRepo := dl.NewInMemoryUploadRepository()
	uploadService := NewUploadService(uploadRepo, logger)

	req := &models.UploadInitiateRequest{
		VideoID:   "vid123",
		UserID:    "user456",
		TotalSize: 100 * 1024 * 1024, // 100 MB
	}

	ctx := context.Background()
	upload, err := uploadService.InitiateUpload(ctx, req)

	if err != nil {
		t.Fatalf("InitiateUpload failed: %v", err)
	}

	if upload.ID == "" {
		t.Error("Upload ID is empty")
	}

	if upload.VideoID != "vid123" {
		t.Errorf("Expected VideoID 'vid123', got '%s'", upload.VideoID)
	}

	if upload.UserID != "user456" {
		t.Errorf("Expected UserID 'user456', got '%s'", upload.UserID)
	}

	if upload.Status != "PENDING" {
		t.Errorf("Expected status 'PENDING', got '%s'", upload.Status)
	}

	if upload.Percentage != 0 {
		t.Errorf("Expected percentage 0, got %f", upload.Percentage)
	}
}

func TestRecordChunkUpload(t *testing.T) {
	logger := log.New(os.Stdout, "[Test] ", log.LstdFlags)
	uploadRepo := dl.NewInMemoryUploadRepository()
	uploadService := NewUploadService(uploadRepo, logger)

	// Initiate upload
	req := &models.UploadInitiateRequest{
		VideoID:   "vid123",
		UserID:    "user456",
		TotalSize: 10 * 1024 * 1024, // 10 MB
	}

	ctx := context.Background()
	upload, err := uploadService.InitiateUpload(ctx, req)
	if err != nil {
		t.Fatalf("InitiateUpload failed: %v", err)
	}

	uploadID := upload.ID

	// Record chunk uploads
	chunkSize := int64(5 * 1024 * 1024) // 5 MB
	upload, err = uploadService.RecordChunkUpload(ctx, uploadID, chunkSize)
	if err != nil {
		t.Fatalf("RecordChunkUpload failed: %v", err)
	}

	if upload.UploadedSize != chunkSize {
		t.Errorf("Expected uploaded size %d, got %d", chunkSize, upload.UploadedSize)
	}

	if upload.UploadedChunks != 1 {
		t.Errorf("Expected uploaded chunks 1, got %d", upload.UploadedChunks)
	}

	expectedPercentage := (float64(chunkSize) / float64(10*1024*1024)) * 100
	if upload.Percentage < expectedPercentage-1 || upload.Percentage > expectedPercentage+1 {
		t.Errorf("Expected percentage ~%f, got %f", expectedPercentage, upload.Percentage)
	}

	if upload.Status != "IN_PROGRESS" {
		t.Errorf("Expected status 'IN_PROGRESS', got '%s'", upload.Status)
	}

	// Record second chunk
	upload, err = uploadService.RecordChunkUpload(ctx, uploadID, chunkSize)
	if err != nil {
		t.Fatalf("RecordChunkUpload failed: %v", err)
	}

	if upload.UploadedSize != 2*chunkSize {
		t.Errorf("Expected uploaded size %d, got %d", 2*chunkSize, upload.UploadedSize)
	}

	if upload.UploadedChunks != 2 {
		t.Errorf("Expected uploaded chunks 2, got %d", upload.UploadedChunks)
	}
}

func TestCompleteUpload(t *testing.T) {
	logger := log.New(os.Stdout, "[Test] ", log.LstdFlags)
	uploadRepo := dl.NewInMemoryUploadRepository()
	uploadService := NewUploadService(uploadRepo, logger)

	req := &models.UploadInitiateRequest{
		VideoID:   "vid123",
		UserID:    "user456",
		TotalSize: 10 * 1024 * 1024,
	}

	ctx := context.Background()
	upload, err := uploadService.InitiateUpload(ctx, req)
	if err != nil {
		t.Fatalf("InitiateUpload failed: %v", err)
	}

	uploadID := upload.ID

	// Complete upload
	upload, err = uploadService.CompleteUpload(ctx, uploadID)
	if err != nil {
		t.Fatalf("CompleteUpload failed: %v", err)
	}

	if upload.Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", upload.Status)
	}

	if upload.Percentage != 100 {
		t.Errorf("Expected percentage 100, got %f", upload.Percentage)
	}

	if upload.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
}

func TestListUserUploads(t *testing.T) {
	logger := log.New(os.Stdout, "[Test] ", log.LstdFlags)
	uploadRepo := dl.NewInMemoryUploadRepository()
	uploadService := NewUploadService(uploadRepo, logger)

	userID := "user456"
	ctx := context.Background()

	// Create multiple uploads for the same user
	for i := 0; i < 3; i++ {
		req := &models.UploadInitiateRequest{
			VideoID:   "vid" + string(rune(i)),
			UserID:    userID,
			TotalSize: 10 * 1024 * 1024,
		}
		_, err := uploadService.InitiateUpload(ctx, req)
		if err != nil {
			t.Fatalf("InitiateUpload failed: %v", err)
		}
	}

	// List uploads for user
	uploads, err := uploadService.ListUserUploads(ctx, userID)
	if err != nil {
		t.Fatalf("ListUserUploads failed: %v", err)
	}

	if len(uploads) != 3 {
		t.Errorf("Expected 3 uploads, got %d", len(uploads))
	}

	// Verify all uploads belong to the user
	for _, upload := range uploads {
		if upload.UserID != userID {
			t.Errorf("Expected UserID '%s', got '%s'", userID, upload.UserID)
		}
	}
}
