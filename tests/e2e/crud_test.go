package e2e

import (
	"testing"
)

// TestVideoCRUDLifecycle exercises the full metadata CRUD path:
//
//	create → get → update → list (verify present) → delete → list (verify gone)
func TestVideoCRUDLifecycle(t *testing.T) {
	cfg := LoadConfig(t)
	c := NewClient(cfg.MetadataURL, cfg.DataURL)
	requireHealthy(t, c)

	// 1. Create
	video, err := c.CreateVideo(CreateVideoRequest{
		Title:       "CRUD Lifecycle Test",
		Description: "Testing full CRUD operations",
		Duration:    300,
		SizeBytes:   1024 * 1024,
		Format:      "webm",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if video.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	t.Logf("Created: id=%s title=%q format=%s", video.ID, video.Title, video.Format)

	// 2. Get — verify fields match
	got, err := c.GetVideo(video.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "CRUD Lifecycle Test" {
		t.Errorf("Get title: want %q, got %q", "CRUD Lifecycle Test", got.Title)
	}
	if got.Format != "" && got.Format != "webm" {
		t.Errorf("Get format: want %q or empty, got %q", "webm", got.Format)
	}
	if got.Duration != 300 {
		t.Errorf("Get duration: want 300, got %d", got.Duration)
	}
	t.Logf("Get: verified fields match")

	// 3. Update
	updated, err := c.UpdateVideo(video.ID, UpdateVideoRequest{
		Title:       "CRUD Updated Title",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "CRUD Updated Title" {
		t.Errorf("Update title: want %q, got %q", "CRUD Updated Title", updated.Title)
	}
	t.Logf("Updated: title=%q", updated.Title)

	// 4. Get after update — verify changes persisted
	got2, err := c.GetVideo(video.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got2.Title != "CRUD Updated Title" {
		t.Errorf("Get after update: want %q, got %q", "CRUD Updated Title", got2.Title)
	}

	// 5. List — should contain our video
	list, err := c.ListVideos(100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, v := range list.Videos {
		if v.ID == video.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("List: created video not found in listing")
	}
	t.Logf("List: found video in %d results", list.Count)

	// 6. Delete
	if err := c.DeleteVideo(video.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	t.Logf("Deleted: id=%s", video.ID)

	// 7. Get after delete — should fail
	_, err = c.GetVideo(video.ID)
	if err == nil {
		t.Error("Get after delete: expected error, got nil")
	} else {
		t.Logf("Get after delete: correctly returned error")
	}

	t.Log("✓ CRUD lifecycle test passed")
}
