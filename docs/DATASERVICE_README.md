# Data Service

The Data Service is a gRPC-based microservice responsible for managing upload sessions and tracking upload progress. It maintains state for all active uploads and provides APIs for upload lifecycle management.

## Architecture

The Data Service follows a clean layered architecture:

- **Models**: Data structures for upload sessions and API requests/responses
- **Repository**: Data access layer using in-memory storage
- **Service**: Business logic for upload management
- **Server**: gRPC service implementation
- **Proto**: gRPC service definitions

## Features

- **Upload Initiation**: Start new upload sessions with metadata
- **Progress Tracking**: Monitor upload progress with speed and ETA
- **Chunk Management**: Track individual chunk uploads
- **Upload Completion**: Mark uploads as completed
- **Upload Listing**: Retrieve all uploads for a user
- **Error Handling**: Comprehensive error handling and logging

## Project Structure

```
dataservice/
├── main.go                 # Service entry point
├── models/                 # Data models
│   └── upload.go          # Upload models
├── repository/            # Data access layer
│   ├── interfaces.go      # Repository interfaces
│   └── memory.go          # In-memory implementation
├── service/               # Business logic
│   ├── upload_service.go  # Upload service implementation
│   └── upload_service_test.go # Service tests
├── server/                # gRPC server
│   └── grpc_server.go     # gRPC service implementation
├── proto/                 # Protocol buffers
│   └── dataservice.proto  # gRPC service definition
├── pb/                    # Generated proto code
│   ├── dataservice.pb.go           # Message definitions
│   ├── dataservice_grpc.pb.go      # gRPC server/client code
│   └── messages.go                 # Additional message types
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Protocol Buffer compiler (for regenerating proto files)

### Building

```bash
cd dataservice
go mod download
go build -o data-service main.go
```

### Running

Start the Data Service gRPC server:

```bash
./data-service -port 50051
```

The server will listen on port 50051 (default).

### Running Tests

```bash
go test ./...
```

## API Reference

### InitiateUpload

Starts a new upload session.

```protobuf
rpc InitiateUpload(UploadInitiateRequest) returns (UploadInitiateResponse);

message UploadInitiateRequest {
  string video_id = 1;
  string user_id = 2;
  int64 total_size = 3;
}

message UploadInitiateResponse {
  string upload_id = 1;
  int64 chunk_size = 2;
  string message = 3;
}
```

### UploadChunk

Uploads a chunk of data.

```protobuf
rpc UploadChunk(UploadChunkRequest) returns (UploadChunkResponse);

message UploadChunkRequest {
  string upload_id = 1;
  int32 chunk_index = 2;
  int32 total_chunks = 3;
  string checksum = 4;
  bytes chunk_data = 5;
}

message UploadChunkResponse {
  string upload_id = 1;
  int32 chunk_index = 2;
  bool success = 3;
  string message = 4;
}
```

### GetUploadProgress

Retrieves the current progress of an upload.

```protobuf
rpc GetUploadProgress(UploadProgressRequest) returns (UploadProgressResponse);

message UploadProgressRequest {
  string upload_id = 1;
}

message UploadProgressResponse {
  string upload_id = 1;
  double percentage = 2;
  int64 uploaded_bytes = 3;
  int64 total_bytes = 4;
  int32 uploaded_chunks = 5;
  int32 total_chunks = 6;
  double speed_mbps = 7;
  int64 estimated_seconds = 8;
}
```

### CompleteUpload

Marks an upload as completed.

```protobuf
rpc CompleteUpload(CompleteUploadRequest) returns (CompleteUploadResponse);

message CompleteUploadRequest {
  string upload_id = 1;
  string final_checksum = 2;
}

message CompleteUploadResponse {
  string upload_id = 1;
  string status = 2;
  string message = 3;
}
```

### ListUploads

Lists all uploads for a user.

```protobuf
rpc ListUploads(ListUploadsRequest) returns (ListUploadsResponse);

message ListUploadsRequest {
  string user_id = 1;
}

message ListUploadsResponse {
  repeated Upload uploads = 1;
}
```

## Configuration

Environment variables:
- `PORT` (default: 50051): gRPC server port

## Storage

The Data Service currently uses in-memory storage. This is suitable for development and testing. For production:

1. Implement `DatabaseUploadRepository` using a persistent database
2. Update `main.go` to use the database implementation
3. Add database configuration and connection management

## Monitoring and Logging

The service logs all significant events:
- Upload initiations
- Chunk uploads
- Progress updates
- Errors and failures

Logs are written to stdout with timestamp and severity information.

## Future Enhancements

1. Persistent database storage
2. Metrics collection (Prometheus)
3. Distributed tracing
4. Rate limiting and quota management
5. Cleanup of stale uploads
6. Upload resumption support
