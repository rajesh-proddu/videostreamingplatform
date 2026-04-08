# Data Service Quick Start Guide

## Prerequisites

- Go 1.21 or later
- Docker (optional)

## Building the Service

### Option 1: Using Make
```bash
cd dataservice
make build
```

### Option 2: Manual Build
```bash
cd dataservice
go mod tidy
go build -o data-service main.go
```

## Running the Service

### Option 1: Local Binary
```bash
./data-service -port 50051
```

### Option 2: Using Docker
```bash
docker build -t data-service:latest .
docker run -p 50051:50051 data-service:latest
```

### Option 3: Using Make
```bash
make run
```

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Or using Make
make test
make test-verbose
```

## Expected Output

When the service starts:
```
[DataService] 2026/04/08 05:15:49 Data Service gRPC server listening on port 50051
```

## API Testing

### Using grpcurl

```bash
# Test InitiateUpload
grpcurl -plaintext \
  -d '{"video_id":"vid123","user_id":"user456","total_size":1048576}' \
  localhost:50051 \
  dataservice.DataService/InitiateUpload

# Test GetUploadProgress
grpcurl -plaintext \
  -d '{"upload_id":"upl_123456"}' \
  localhost:50051 \
  dataservice.DataService/GetUploadProgress
```

### Using Go Client

```go
package main

import (
	"context"
	"log"
	
	"github.com/yourusername/videostreamingplatform/dataservice/pb"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	client := pb.NewDataServiceClient(conn)
	
	// Initiate upload
	resp, err := client.InitiateUpload(context.Background(), &pb.UploadInitiateRequest{
		VideoId:   "vid123",
		UserId:    "user456",
		TotalSize: 1048576,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	
	log.Printf("Upload ID: %s", resp.UploadId)
}
```

## Project Structure

```
dataservice/
├── Makefile                    # Build commands
├── Dockerfile                  # Docker build file
├── README.md                   # Full documentation
├── main.go                     # Service entry point
├── models/                     # Data models
├── repository/                 # Data access layer
├── service/                    # Business logic
├── server/                     # gRPC implementation
├── proto/                      # Protocol definitions
└── pb/                         # Generated proto code
```

## Troubleshooting

### Build Issues

**Error**: `package videostreamingplatform is not in GOPATH`
- Solution: Ensure the module name in `go.mod` matches your imports

**Error**: `protobuf: syntax error`
- Solution: Verify the `.proto` file syntax and regenerate with protoc

### Runtime Issues

**Error**: `Address already in use`
- Solution: Change the port with `-port` flag or kill the process using port 50051

**Error**: `Connection refused`
- Solution: Ensure the service is running on the correct port

## Available Make Targets

```bash
make help           # Show available targets
make build          # Build binary
make run            # Build and run
make test           # Run tests
make test-verbose   # Run tests with verbose output
make clean          # Clean build artifacts
make proto          # Show proto file location
```

## Logs

The service logs all operations to stdout with format:
```
[DataService] YYYY/MM/DD HH:MM:SS filename:line message
```

## Configuration

Currently, the service supports:
- **Port**: Use `-port {number}` flag (default: 50051)

Future configuration options:
- Database connection string
- Logging level
- Storage backend

## Next Steps

1. integrate with API Gateway
2. Add database persistence
3. Implement authentication
4. Set up monitoring and metrics
5. Deploy to Kubernetes

## Support

Refer to [README.md](./README.md) for detailed documentation.
