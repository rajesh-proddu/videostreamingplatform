// Package handlers provides HTTP request handlers for the data service
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/yourusername/videostreamingplatform/dataservice/bl"
	"github.com/yourusername/videostreamingplatform/dataservice/models"
	"github.com/yourusername/videostreamingplatform/dataservice/storage"
)

// UploadHandler handles HTTP requests for upload operations
type UploadHandler struct {
	service *bl.UploadService
	storage *storage.S3Client
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(service *bl.UploadService, s3 *storage.S3Client) *UploadHandler {
	return &UploadHandler{
		service: service,
		storage: s3,
	}
}

// InitiateUpload handles upload initiation requests
func (h *UploadHandler) InitiateUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.UploadInitiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.VideoID == "" || req.UserID == "" || req.TotalSize == 0 {
		http.Error(w, "video_id, user_id, and total_size are required", http.StatusBadRequest)
		return
	}

	upload, err := h.service.InitiateUpload(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to initiate upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(models.UploadInitiateResponse{
		UploadID:  upload.ID,
		ChunkSize: 5 * 1024 * 1024,
		Message:   "Upload initiated",
	})
}

// Upload handles chunk uploads to S3 and tracks progress
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.PathValue("uploadId")
	if uploadID == "" {
		http.Error(w, "Upload ID is required", http.StatusBadRequest)
		return
	}

	chunkIndexStr := r.URL.Query().Get("chunkIndex")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		http.Error(w, "Invalid chunkIndex", http.StatusBadRequest)
		return
	}

	// Read chunk data from request body
	chunkData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read chunk data", http.StatusBadRequest)
		return
	}

	// Get upload session to find video ID
	upload, err := h.service.GetUploadProgress(r.Context(), uploadID)
	if err != nil {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	// Upload chunk to S3
	key := fmt.Sprintf("videos/%s/chunk_%d", upload.VideoID, chunkIndex)
	if err := h.storage.Upload(r.Context(), key, bytes.NewReader(chunkData), int64(len(chunkData))); err != nil {
		http.Error(w, "Failed to upload chunk to storage", http.StatusInternalServerError)
		return
	}

	// Record chunk in the upload tracker
	if _, err := h.service.RecordChunkUpload(r.Context(), uploadID, int64(len(chunkData))); err != nil {
		http.Error(w, "Failed to record chunk upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "video_id": upload.VideoID})
}

// Download handles file downloads from S3
func (h *UploadHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoID := r.PathValue("id")
	if videoID == "" {
		http.Error(w, "Video ID is required", http.StatusBadRequest)
		return
	}

	// Download from S3
	key := "videos/" + videoID
	body, err := h.storage.Download(r.Context(), key)
	if err != nil {
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	defer func() { _ = body.Close() }()

	// Stream to client
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "attachment; filename=video.mp4")
	if _, err := io.Copy(w, body); err != nil {
		http.Error(w, "Failed to stream video", http.StatusInternalServerError)
		return
	}
}

// GetUploadProgress retrieves the progress of an upload
func (h *UploadHandler) GetUploadProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.PathValue("uploadId")
	if uploadID == "" {
		http.Error(w, "Upload ID is required", http.StatusBadRequest)
		return
	}

	upload, err := h.service.GetUploadProgress(r.Context(), uploadID)
	if err != nil {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(upload)
}

// CompleteUpload finalizes an upload and merges chunks into a single S3 object
func (h *UploadHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.PathValue("uploadId")
	if uploadID == "" {
		http.Error(w, "Upload ID is required", http.StatusBadRequest)
		return
	}

	// Get upload to find video ID and total chunks
	progress, err := h.service.GetUploadProgress(r.Context(), uploadID)
	if err != nil {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	// Merge chunks into a single S3 object
	var merged bytes.Buffer
	for i := 0; i < progress.UploadedChunks; i++ {
		chunkKey := fmt.Sprintf("videos/%s/chunk_%d", progress.VideoID, i)
		body, err := h.storage.Download(r.Context(), chunkKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read chunk %d", i), http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(&merged, body); err != nil {
			_ = body.Close()
			http.Error(w, fmt.Sprintf("Failed to copy chunk %d", i), http.StatusInternalServerError)
			return
		}
		_ = body.Close()
	}

	// Upload merged file
	finalKey := fmt.Sprintf("videos/%s", progress.VideoID)
	mergedBytes := merged.Bytes()
	if err := h.storage.Upload(r.Context(), finalKey, bytes.NewReader(mergedBytes), int64(len(mergedBytes))); err != nil {
		http.Error(w, "Failed to write merged file", http.StatusInternalServerError)
		return
	}

	// Clean up chunk objects
	for i := 0; i < progress.UploadedChunks; i++ {
		chunkKey := fmt.Sprintf("videos/%s/chunk_%d", progress.VideoID, i)
		_ = h.storage.Delete(r.Context(), chunkKey)
	}

	upload, err := h.service.CompleteUpload(r.Context(), uploadID)
	if err != nil {
		http.Error(w, "Failed to complete upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models.CompleteUploadResponse{
		UploadID: upload.ID,
		Status:   upload.Status,
		Message:  "Upload completed",
	})
}
