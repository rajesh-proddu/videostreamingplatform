// Package handlers provides HTTP request handlers for the metadata service
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/yourusername/videostreamingplatform/metadataservice/bl"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

// VideoHandler handles HTTP requests for video operations
type VideoHandler struct {
	service *bl.VideoService
}

// NewVideoHandler creates a new video handler
func NewVideoHandler(service *bl.VideoService) *VideoHandler {
	return &VideoHandler{service: service}
}

// CreateVideo creates a new video metadata entry
// @Summary      Create a video
// @Description  Creates a new video metadata record
// @Tags         videos
// @Accept       json
// @Produce      json
// @Param        body  body      models.CreateVideoRequest  true  "Video metadata"
// @Success      201   {object}  models.Video
// @Failure      400   {string}  string  "Invalid request body"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /videos [post]
func (h *VideoHandler) CreateVideo(w http.ResponseWriter, r *http.Request) {
	var req models.CreateVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	video, err := h.service.CreateVideo(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(video)
}

// GetVideo retrieves a video by ID
// @Summary      Get a video
// @Description  Retrieves video metadata by ID
// @Tags         videos
// @Produce      json
// @Param        id   path      string  true  "Video ID"
// @Success      200  {object}  models.Video
// @Failure      404  {string}  string  "Video not found"
// @Router       /videos/{id} [get]
func (h *VideoHandler) GetVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	video, err := h.service.GetVideo(r.Context(), id)
	if err != nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(video)
}

// UpdateVideo updates a video's metadata
// @Summary      Update a video
// @Description  Updates an existing video's metadata
// @Tags         videos
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "Video ID"
// @Param        body  body      models.UpdateVideoRequest  true  "Updated fields"
// @Success      200   {object}  models.Video
// @Failure      400   {string}  string  "Invalid request body"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /videos/{id} [put]
func (h *VideoHandler) UpdateVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req models.UpdateVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	video, err := h.service.UpdateVideo(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(video)
}

// DeleteVideo deletes a video
// @Summary      Delete a video
// @Description  Deletes a video by ID
// @Tags         videos
// @Param        id   path  string  true  "Video ID"
// @Success      204  "No Content"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /videos/{id} [delete]
func (h *VideoHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.service.DeleteVideo(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListVideos lists all videos
// @Summary      List videos
// @Description  Lists all videos with pagination
// @Tags         videos
// @Produce      json
// @Param        limit   query     int  false  "Max results (default 20)"
// @Param        offset  query     int  false  "Offset for pagination"
// @Success      200     {object}  map[string]interface{}
// @Failure      500     {string}  string  "Internal server error"
// @Router       /videos [get]
func (h *VideoHandler) ListVideos(w http.ResponseWriter, r *http.Request) {
	var limit, offset int

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	if limit == 0 {
		limit = 20
	}

	videos, err := h.service.ListVideos(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"videos": videos,
		"count":  len(videos),
	})
}
