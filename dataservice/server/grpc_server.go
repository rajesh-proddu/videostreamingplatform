// Package server provides gRPC server implementations
package server

import (
	"context"
	"log"

	"github.com/yourusername/videostreamingplatform/dataservice/bl"
	"github.com/yourusername/videostreamingplatform/dataservice/models"
	"github.com/yourusername/videostreamingplatform/dataservice/pb"
)

// DataServiceServer implements the gRPC DataService
type DataServiceServer struct {
	pb.UnimplementedDataServiceServer
	uploadService *bl.UploadService
	logger        *log.Logger
}

// NewDataServiceServer creates a new DataService gRPC server
func NewDataServiceServer(uploadService *bl.UploadService, logger *log.Logger) *DataServiceServer {
	return &DataServiceServer{
		uploadService: uploadService,
		logger:        logger,
	}
}

// InitiateUpload initiates a new upload
func (s *DataServiceServer) InitiateUpload(ctx context.Context, req *pb.UploadInitiateRequest) (*pb.UploadInitiateResponse, error) {
	serviceReq := &models.UploadInitiateRequest{
		VideoID:   req.VideoId,
		UserID:    req.UserId,
		TotalSize: req.TotalSize,
	}

	upload, err := s.uploadService.InitiateUpload(ctx, serviceReq)
	if err != nil {
		s.logger.Printf("Error initiating upload: %v", err)
		return nil, err
	}

	return &pb.UploadInitiateResponse{
		UploadId:  upload.ID,
		ChunkSize: 5 * 1024 * 1024, // 5 MB chunks
		Message:   "Upload initiated successfully",
	}, nil
}

// UploadChunk handles chunk uploads
func (s *DataServiceServer) UploadChunk(ctx context.Context, req *pb.UploadChunkRequest) (*pb.UploadChunkResponse, error) {
	chunkSize := int64(len(req.ChunkData))

	_, err := s.uploadService.RecordChunkUpload(ctx, req.UploadId, chunkSize)
	if err != nil {
		s.logger.Printf("Error recording chunk upload: %v", err)
		return nil, err
	}

	return &pb.UploadChunkResponse{
		UploadId:   req.UploadId,
		ChunkIndex: req.ChunkIndex,
		Success:    true,
		Message:    "Chunk uploaded successfully",
	}, nil
}

// GetUploadProgress gets the progress of an upload
func (s *DataServiceServer) GetUploadProgress(ctx context.Context, req *pb.UploadProgressRequest) (*pb.UploadProgressResponse, error) {
	upload, err := s.uploadService.GetUploadProgress(ctx, req.UploadId)
	if err != nil {
		s.logger.Printf("Error getting upload progress: %v", err)
		return nil, err
	}

	estimatedSeconds := int64(0)
	if upload.EstimatedSeconds != nil {
		estimatedSeconds = *upload.EstimatedSeconds
	}

	return &pb.UploadProgressResponse{
		UploadId:         req.UploadId,
		Percentage:       upload.Percentage,
		UploadedBytes:    upload.UploadedSize,
		TotalBytes:       upload.TotalSize,
		UploadedChunks:   int32(upload.UploadedChunks),
		TotalChunks:      int32(upload.TotalChunks),
		SpeedMbps:        upload.SpeedMbps,
		EstimatedSeconds: estimatedSeconds,
	}, nil
}

// CompleteUpload completes an upload
func (s *DataServiceServer) CompleteUpload(ctx context.Context, req *pb.CompleteUploadRequest) (*pb.CompleteUploadResponse, error) {
	upload, err := s.uploadService.CompleteUpload(ctx, req.UploadId)
	if err != nil {
		s.logger.Printf("Error completing upload: %v", err)
		return nil, err
	}

	return &pb.CompleteUploadResponse{
		UploadId: req.UploadId,
		Status:   upload.Status,
		Message:  "Upload completed successfully",
	}, nil
}

// ListUploads lists all uploads for a user
func (s *DataServiceServer) ListUploads(ctx context.Context, req *pb.ListUploadsRequest) (*pb.ListUploadsResponse, error) {
	uploads, err := s.uploadService.ListUserUploads(ctx, req.UserId)
	if err != nil {
		s.logger.Printf("Error listing uploads: %v", err)
		return nil, err
	}

	pbUploads := make([]*pb.Upload, 0, len(uploads))
	for _, u := range uploads {
		pbUploads = append(pbUploads, s.modelToProto(u))
	}

	return &pb.ListUploadsResponse{
		Uploads: pbUploads,
	}, nil
}

// Helper function to convert model to proto
func (s *DataServiceServer) modelToProto(u *models.Upload) *pb.Upload {
	completedAt := ""
	if u.CompletedAt != nil {
		completedAt = u.CompletedAt.String()
	}

	return &pb.Upload{
		Id:             u.ID,
		VideoId:        u.VideoID,
		UserId:         u.UserID,
		TotalSize:      u.TotalSize,
		UploadedSize:   u.UploadedSize,
		UploadedChunks: int32(u.UploadedChunks),
		TotalChunks:    int32(u.TotalChunks),
		Status:         u.Status,
		Percentage:     u.Percentage,
		SpeedMbps:      u.SpeedMbps,
		CreatedAt:      u.CreatedAt.String(),
		UpdatedAt:      u.UpdatedAt.String(),
		CompletedAt:    completedAt,
	}
}
