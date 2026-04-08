package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yourusername/videostreamingplatform/metadataservice/models"
)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MySQL{db: db}, nil
}

func (m *MySQL) Close() error {
	return m.db.Close()
}

func (m *MySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// CreateVideo creates a new video record
func (m *MySQL) CreateVideo(ctx context.Context, video *models.Video) error {
	query := `INSERT INTO videos (id, title, description, duration, size_bytes, upload_status) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err := m.db.ExecContext(ctx, query, video.ID, video.Title, video.Description,
		video.Duration, video.SizeBytes, "PENDING")
	return err
}

// GetVideo retrieves a video by ID
func (m *MySQL) GetVideo(ctx context.Context, id string) (*models.Video, error) {
	query := `SELECT id, title, description, duration, size_bytes, upload_status, created_at, updated_at 
	          FROM videos WHERE id = ?`
	row := m.db.QueryRowContext(ctx, query, id)

	video := &models.Video{}
	err := row.Scan(&video.ID, &video.Title, &video.Description, &video.Duration,
		&video.SizeBytes, &video.UploadStatus, &video.CreatedAt, &video.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return video, nil
}

// UpdateVideo updates video metadata
func (m *MySQL) UpdateVideo(ctx context.Context, video *models.Video) error {
	query := `UPDATE videos SET title = ?, description = ?, duration = ?, size_bytes = ?, 
	          upload_status = ? WHERE id = ?`
	_, err := m.db.ExecContext(ctx, query, video.Title, video.Description, video.Duration,
		video.SizeBytes, video.UploadStatus, video.ID)
	return err
}

// DeleteVideo deletes a video record
func (m *MySQL) DeleteVideo(ctx context.Context, id string) error {
	query := `DELETE FROM videos WHERE id = ?`
	_, err := m.db.ExecContext(ctx, query, id)
	return err
}

// ListVideos lists all videos
func (m *MySQL) ListVideos(ctx context.Context, limit, offset int) ([]*models.Video, error) {
	query := `SELECT id, title, description, duration, size_bytes, upload_status, created_at, updated_at 
	          FROM videos ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := m.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*models.Video
	for rows.Next() {
		video := &models.Video{}
		err := rows.Scan(&video.ID, &video.Title, &video.Description, &video.Duration,
			&video.SizeBytes, &video.UploadStatus, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}
	return videos, nil
}

// CreateUpload creates a new upload record
func (m *MySQL) CreateUpload(ctx context.Context, upload *models.Upload) error {
	query := `INSERT INTO uploads (id, video_id, user_id, status) VALUES (?, ?, ?, ?)`
	_, err := m.db.ExecContext(ctx, query, upload.ID, upload.VideoID, upload.UserID, upload.Status)
	return err
}

// UpdateUploadStatus updates upload progress
func (m *MySQL) UpdateUploadStatus(ctx context.Context, uploadID string, status string, completedChunks int) error {
	query := `UPDATE uploads SET status = ?, completed_chunks = ?, updated_at = NOW() WHERE id = ?`
	_, err := m.db.ExecContext(ctx, query, status, completedChunks, uploadID)
	return err
}
