// Package dl provides repository implementations
package dl

import (
	"context"
	"fmt"
	"sync"

	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

// InMemoryUploadRepository is an in-memory implementation of UploadRepository
type InMemoryUploadRepository struct {
	uploads map[string]*models.Upload
	mu      sync.RWMutex
}

// NewInMemoryUploadRepository creates a new in-memory upload repository
func NewInMemoryUploadRepository() UploadRepository {
	return &InMemoryUploadRepository{
		uploads: make(map[string]*models.Upload),
	}
}

// CreateUpload creates a new upload session
func (r *InMemoryUploadRepository) CreateUpload(ctx context.Context, upload *models.Upload) error {
	if upload == nil {
		return fmt.Errorf("upload cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.uploads[upload.ID]; exists {
		return fmt.Errorf("upload with ID %s already exists", upload.ID)
	}

	r.uploads[upload.ID] = upload
	return nil
}

// GetUploadByID retrieves an upload by ID
func (r *InMemoryUploadRepository) GetUploadByID(ctx context.Context, uploadID string) (*models.Upload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	upload, exists := r.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("upload with ID %s not found", uploadID)
	}

	return upload, nil
}

// ListUploadsByUserID lists all uploads for a user
func (r *InMemoryUploadRepository) ListUploadsByUserID(ctx context.Context, userID string) ([]*models.Upload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var uploads []*models.Upload
	for _, upload := range r.uploads {
		if upload.UserID == userID {
			uploads = append(uploads, upload)
		}
	}

	return uploads, nil
}

// UpdateUpload updates an existing upload
func (r *InMemoryUploadRepository) UpdateUpload(ctx context.Context, upload *models.Upload) error {
	if upload == nil {
		return fmt.Errorf("upload cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.uploads[upload.ID]; !exists {
		return fmt.Errorf("upload with ID %s not found", upload.ID)
	}

	r.uploads[upload.ID] = upload
	return nil
}

// DeleteUpload deletes an upload session
func (r *InMemoryUploadRepository) DeleteUpload(ctx context.Context, uploadID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.uploads[uploadID]; !exists {
		return fmt.Errorf("upload with ID %s not found", uploadID)
	}

	delete(r.uploads, uploadID)
	return nil
}
