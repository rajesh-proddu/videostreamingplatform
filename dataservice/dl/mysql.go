package dl

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yourusername/videostreamingplatform/dataservice/models"
)

// MySQLUploadRepository is a MySQL-backed implementation of UploadRepository.
type MySQLUploadRepository struct {
	db *sql.DB
}

// NewMySQLUploadRepository creates a new MySQL upload repository.
func NewMySQLUploadRepository(db *sql.DB) UploadRepository {
	return &MySQLUploadRepository{db: db}
}

// CreateUpload inserts a new upload record.
func (r *MySQLUploadRepository) CreateUpload(ctx context.Context, upload *models.Upload) error {
	if upload == nil {
		return fmt.Errorf("upload cannot be nil")
	}

	query := `INSERT INTO uploads (id, video_id, user_id, total_size, uploaded_size,
		uploaded_chunks, total_chunks, status, percentage, speed_mbps, estimated_seconds, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		upload.ID, upload.VideoID, upload.UserID,
		upload.TotalSize, upload.UploadedSize,
		upload.UploadedChunks, upload.TotalChunks,
		upload.Status, upload.Percentage,
		nullFloat64(upload.SpeedMbps), nullInt64(upload.EstimatedSeconds),
		upload.CompletedAt,
	)
	return err
}

// GetUploadByID retrieves an upload by ID.
func (r *MySQLUploadRepository) GetUploadByID(ctx context.Context, uploadID string) (*models.Upload, error) {
	query := `SELECT id, video_id, user_id, total_size, uploaded_size, uploaded_chunks, total_chunks,
		status, percentage, speed_mbps, estimated_seconds, created_at, updated_at, completed_at
		FROM uploads WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, uploadID)
	return scanUpload(row)
}

// ListUploadsByUserID retrieves all uploads for a given user.
func (r *MySQLUploadRepository) ListUploadsByUserID(ctx context.Context, userID string) ([]*models.Upload, error) {
	query := `SELECT id, video_id, user_id, total_size, uploaded_size, uploaded_chunks, total_chunks,
		status, percentage, speed_mbps, estimated_seconds, created_at, updated_at, completed_at
		FROM uploads WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var uploads []*models.Upload
	for rows.Next() {
		u, err := scanUploadFromRows(rows)
		if err != nil {
			return nil, err
		}
		uploads = append(uploads, u)
	}
	return uploads, rows.Err()
}

// UpdateUpload updates an existing upload record.
func (r *MySQLUploadRepository) UpdateUpload(ctx context.Context, upload *models.Upload) error {
	if upload == nil {
		return fmt.Errorf("upload cannot be nil")
	}

	query := `UPDATE uploads SET video_id = ?, user_id = ?, total_size = ?, uploaded_size = ?,
		uploaded_chunks = ?, total_chunks = ?, status = ?, percentage = ?,
		speed_mbps = ?, estimated_seconds = ?, completed_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		upload.VideoID, upload.UserID,
		upload.TotalSize, upload.UploadedSize,
		upload.UploadedChunks, upload.TotalChunks,
		upload.Status, upload.Percentage,
		nullFloat64(upload.SpeedMbps), nullInt64(upload.EstimatedSeconds),
		upload.CompletedAt, upload.ID,
	)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("upload with ID %s not found", upload.ID)
	}
	return nil
}

// DeleteUpload deletes an upload record by ID.
func (r *MySQLUploadRepository) DeleteUpload(ctx context.Context, uploadID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM uploads WHERE id = ?`, uploadID)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("upload with ID %s not found", uploadID)
	}
	return nil
}

// scanUpload scans a single row into an Upload model.
func scanUpload(row *sql.Row) (*models.Upload, error) {
	u := &models.Upload{}
	var speedMbps sql.NullFloat64
	var estimatedSeconds sql.NullInt64

	err := row.Scan(
		&u.ID, &u.VideoID, &u.UserID,
		&u.TotalSize, &u.UploadedSize,
		&u.UploadedChunks, &u.TotalChunks,
		&u.Status, &u.Percentage,
		&speedMbps, &estimatedSeconds,
		&u.CreatedAt, &u.UpdatedAt, &u.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	u.SpeedMbps = speedMbps.Float64
	if estimatedSeconds.Valid {
		v := estimatedSeconds.Int64
		u.EstimatedSeconds = &v
	}
	return u, nil
}

// scanUploadFromRows scans the current rows cursor into an Upload model.
func scanUploadFromRows(rows *sql.Rows) (*models.Upload, error) {
	u := &models.Upload{}
	var speedMbps sql.NullFloat64
	var estimatedSeconds sql.NullInt64

	err := rows.Scan(
		&u.ID, &u.VideoID, &u.UserID,
		&u.TotalSize, &u.UploadedSize,
		&u.UploadedChunks, &u.TotalChunks,
		&u.Status, &u.Percentage,
		&speedMbps, &estimatedSeconds,
		&u.CreatedAt, &u.UpdatedAt, &u.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	u.SpeedMbps = speedMbps.Float64
	if estimatedSeconds.Valid {
		v := estimatedSeconds.Int64
		u.EstimatedSeconds = &v
	}
	return u, nil
}

func nullFloat64(v float64) sql.NullFloat64 {
	if v == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
