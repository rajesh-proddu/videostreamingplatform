// Package dl defines the interfaces for data persistence
package dl

import (
	"context"

	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

// UploadRepository defines operations for upload data persistence
type UploadRepository interface {
	// CreateUpload creates a new upload session
	CreateUpload(ctx context.Context, upload *models.Upload) error

	// GetUploadByID retrieves an upload by ID
	GetUploadByID(ctx context.Context, uploadID string) (*models.Upload, error)

	// ListUploadsByUserID lists all uploads for a user
	ListUploadsByUserID(ctx context.Context, userID string) ([]*models.Upload, error)

	// UpdateUpload updates an existing upload
	UpdateUpload(ctx context.Context, upload *models.Upload) error

	// DeleteUpload deletes an upload session
	DeleteUpload(ctx context.Context, uploadID string) error
}
