// Package handlers provides HTTP request handlers for the data service
package handlers

import (
	"encoding/json"
	"io"
	"net/http"

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
	_ = json.NewEncoder(w).Encode(upload)
}

// Upload handles file uploads to S3
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoID := r.PathValue("id")
	if videoID == "" {
		http.Error(w, "Video ID is required", http.StatusBadRequest)
		return
	}

	// Read file from request body
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

	// Upload to S3
	key := "videos/" + videoID
	if err := h.storage.Upload(r.Context(), key, file, r.ContentLength); err != nil {
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "uploaded", "video_id": videoID})
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

// CompleteUpload finalizes an upload
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

	upload, err := h.service.CompleteUpload(r.Context(), uploadID)
	if err != nil {
		http.Error(w, "Failed to complete upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(upload)
}
