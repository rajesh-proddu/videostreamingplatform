package component

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/videostreamingplatform/metadataservice/bl"
	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/handlers"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

// mockVideoStore implements dl.VideoStore with an in-memory map.
type mockVideoStore struct {
	videos map[string]*models.Video
}

func newMockVideoStore() *mockVideoStore {
	return &mockVideoStore{videos: make(map[string]*models.Video)}
}

func (m *mockVideoStore) CreateVideo(_ context.Context, video *models.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *mockVideoStore) GetVideo(_ context.Context, id string) (*models.Video, error) {
	v, ok := m.videos[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return v, nil
}

func (m *mockVideoStore) UpdateVideo(_ context.Context, video *models.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *mockVideoStore) DeleteVideo(_ context.Context, id string) error {
	delete(m.videos, id)
	return nil
}

func (m *mockVideoStore) ListVideos(_ context.Context, limit, offset int) ([]*models.Video, error) {
	var result []*models.Video
	for _, v := range m.videos {
		result = append(result, v)
	}
	return result, nil
}

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := newMockVideoStore()
	repo := dl.NewVideoRepository(store)
	svc := bl.NewVideoService(repo)
	h := handlers.NewVideoHandler(svc, "")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /videos", h.CreateVideo)
	mux.HandleFunc("GET /videos/{id}", h.GetVideo)
	mux.HandleFunc("PUT /videos/{id}", h.UpdateVideo)
	mux.HandleFunc("DELETE /videos/{id}", h.DeleteVideo)
	mux.HandleFunc("GET /videos", h.ListVideos)

	return httptest.NewServer(mux)
}

func createTestVideo(t *testing.T, baseURL string, title string, size int64) models.Video {
	t.Helper()

	body, _ := json.Marshal(models.CreateVideoRequest{
		Title:     title,
		SizeBytes: size,
	})

	resp, err := http.Post(baseURL+"/videos", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /videos failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /videos status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var video models.Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return video
}

func TestCreateVideo_HTTP(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	video := createTestVideo(t, srv.URL, "Test Video", 1024)

	if video.ID == "" {
		t.Error("video ID should not be empty")
	}
	if video.Title != "Test Video" {
		t.Errorf("Title = %q, want %q", video.Title, "Test Video")
	}
	if video.UploadStatus != "PENDING" {
		t.Errorf("UploadStatus = %q, want %q", video.UploadStatus, "PENDING")
	}
}

func TestCreateVideo_InvalidBody(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/videos", "application/json", bytes.NewReader([]byte(`{invalid`)))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetVideo_HTTP(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	created := createTestVideo(t, srv.URL, "Get Me", 2048)

	resp, err := http.Get(srv.URL + "/videos/" + created.ID)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var video models.Video
	_ = json.NewDecoder(resp.Body).Decode(&video)

	if video.ID != created.ID {
		t.Errorf("ID = %q, want %q", video.ID, created.ID)
	}
	if video.Title != "Get Me" {
		t.Errorf("Title = %q, want %q", video.Title, "Get Me")
	}
}

func TestGetVideo_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/videos/nonexistent-id")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestUpdateVideo_HTTP(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	created := createTestVideo(t, srv.URL, "Original", 512)

	updateBody, _ := json.Marshal(models.UpdateVideoRequest{
		Title: "Updated Title",
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/videos/"+created.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var video models.Video
	_ = json.NewDecoder(resp.Body).Decode(&video)
	if video.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", video.Title, "Updated Title")
	}
}

func TestDeleteVideo_HTTP(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	created := createTestVideo(t, srv.URL, "Delete Me", 256)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/videos/"+created.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestListVideos_HTTP(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createTestVideo(t, srv.URL, "Video 1", 100)
	createTestVideo(t, srv.URL, "Video 2", 200)
	createTestVideo(t, srv.URL, "Video 3", 300)

	resp, err := http.Get(srv.URL + "/videos")
	if err != nil {
		t.Fatalf("GET /videos failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Videos []models.Video `json:"videos"`
		Count  int            `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("count = %d, want 3", result.Count)
	}
	if len(result.Videos) != 3 {
		t.Errorf("videos length = %d, want 3", len(result.Videos))
	}
}
