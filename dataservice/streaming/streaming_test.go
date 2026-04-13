package streaming

import (
	"testing"
)

func TestDefaultChunkSize(t *testing.T) {
	t.Parallel()

	expected := int64(5 * 1024 * 1024) // 5MB
	if DefaultChunkSize != expected {
		t.Errorf("DefaultChunkSize = %d, want %d", DefaultChunkSize, expected)
	}
}

func TestNewUploadSession(t *testing.T) {
	t.Parallel()

	session := NewUploadSession("upload-1", "video-1", 1024*1024, 256*1024)

	if session.UploadID != "upload-1" {
		t.Errorf("UploadID = %q, want %q", session.UploadID, "upload-1")
	}
	if session.VideoID != "video-1" {
		t.Errorf("VideoID = %q, want %q", session.VideoID, "video-1")
	}
	if session.TotalSize != 1024*1024 {
		t.Errorf("TotalSize = %d, want %d", session.TotalSize, 1024*1024)
	}
	if session.ChunkSize != 256*1024 {
		t.Errorf("ChunkSize = %d, want %d", session.ChunkSize, 256*1024)
	}
	if session.ReceivedChunks == nil {
		t.Fatal("ReceivedChunks should be initialized (non-nil)")
	}
	if len(session.ReceivedChunks) != 0 {
		t.Errorf("ReceivedChunks should be empty, got %d entries", len(session.ReceivedChunks))
	}
}

func TestNewUploadSession_ZeroSize(t *testing.T) {
	t.Parallel()

	session := NewUploadSession("upload-2", "video-2", 0, 0)

	if session.TotalSize != 0 {
		t.Errorf("TotalSize = %d, want 0", session.TotalSize)
	}
	if session.ReceivedChunks == nil {
		t.Fatal("ReceivedChunks should be initialized even with zero size")
	}
}
