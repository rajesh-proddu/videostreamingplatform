// Package handlers provides HTTP request handlers for the data service
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/yourusername/videostreamingplatform/dataservice/bl"
	"github.com/yourusername/videostreamingplatform/dataservice/models"
	"github.com/yourusername/videostreamingplatform/dataservice/storage"
	"github.com/yourusername/videostreamingplatform/utils/events"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
	"github.com/yourusername/videostreamingplatform/utils/observability"
)

// UploadHandler handles HTTP requests for upload operations
type UploadHandler struct {
	service       *bl.UploadService
	storage       *storage.S3Client
	watchProducer kafka.Producer
	logger        *log.Logger
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(service *bl.UploadService, s3 *storage.S3Client, watchProducer kafka.Producer, obsLogger *observability.Logger) *UploadHandler {
	var l *log.Logger
	if obsLogger != nil {
		l = obsLogger.Logger
	}
	return &UploadHandler{
		service:       service,
		storage:       s3,
		watchProducer: watchProducer,
		logger:        l,
	}
}

// InitiateUpload handles upload initiation requests
// @Summary      Initiate an upload
// @Description  Creates a new upload session for a video
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        body  body      models.UploadInitiateRequest  true  "Upload initiation"
// @Success      201   {object}  models.UploadInitiateResponse
// @Failure      400   {string}  string  "Invalid request"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /uploads/initiate [post]
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
// @Summary      Upload a chunk
// @Description  Uploads a single chunk of a video file
// @Tags         uploads
// @Accept       application/octet-stream
// @Produce      json
// @Param        uploadId    path   string  true  "Upload session ID"
// @Param        chunkIndex  query  int     true  "Chunk index"
// @Success      200  {object}  map[string]string
// @Failure      400  {string}  string  "Invalid request"
// @Failure      404  {string}  string  "Upload not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /uploads/{uploadId}/chunks [post]
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
// @Summary      Download a video
// @Description  Streams a video file from storage
// @Tags         videos
// @Produce      video/mp4
// @Param        id       path   string  true   "Video ID"
// @Param        user_id  query  string  false  "User ID for watch tracking"
// @Success      200  {file}    video/mp4
// @Failure      400  {string}  string  "Video ID required"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /videos/{id}/download [get]
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

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Header.Get("X-User-ID")
	}

	sessionID := uuid.New().String()

	// Publish watch.started event (best-effort)
	h.publishWatchEvent(r, events.WatchStarted, videoID, userID, sessionID, 0)

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
	bytesWritten, err := io.Copy(w, body)
	if err != nil {
		// Cannot write HTTP error after partial response
		if h.logger != nil {
			h.logger.Printf("Failed to stream video %s: %v", videoID, err)
		}
		return
	}

	// Publish watch.completed event (best-effort)
	h.publishWatchEvent(r, events.WatchCompleted, videoID, userID, sessionID, bytesWritten)
}

// publishWatchEvent publishes a watch event to Kafka (best-effort: logs errors, never fails the request).
func (h *UploadHandler) publishWatchEvent(r *http.Request, eventType, videoID, userID, sessionID string, bytesRead int64) {
	if h.watchProducer == nil {
		return
	}

	evt := events.NewWatchEvent(eventType, events.WatchPayload{
		VideoID:   videoID,
		UserID:    userID,
		SessionID: sessionID,
		BytesRead: bytesRead,
	})

	data, err := evt.Marshal()
	if err != nil {
		if h.logger != nil {
			h.logger.Printf("Failed to marshal watch event: %v", err)
		}
		return
	}

	if err := h.watchProducer.Publish(r.Context(), []byte(videoID), data); err != nil {
		if h.logger != nil {
			h.logger.Printf("Failed to publish %s event for video %s: %v", eventType, videoID, err)
		}
	}
}

// GetUploadProgress retrieves the progress of an upload
// @Summary      Get upload progress
// @Description  Returns the current progress of an upload session
// @Tags         uploads
// @Produce      json
// @Param        uploadId  path      string  true  "Upload session ID"
// @Success      200       {object}  models.Upload
// @Failure      400       {string}  string  "Upload ID required"
// @Failure      404       {string}  string  "Upload not found"
// @Router       /uploads/{uploadId}/progress [get]
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
// @Summary      Complete an upload
// @Description  Merges uploaded chunks and finalizes the upload session
// @Tags         uploads
// @Produce      json
// @Param        uploadId  path      string  true  "Upload session ID"
// @Success      200       {object}  models.CompleteUploadResponse
// @Failure      400       {string}  string  "Upload ID required"
// @Failure      404       {string}  string  "Upload not found"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /uploads/{uploadId}/complete [post]
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
