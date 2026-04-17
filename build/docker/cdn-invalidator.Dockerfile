# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cdn-invalidator ./workers/cdn-invalidator

# Runtime stage
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 videostreamingplatform && \
    adduser -D -u 1000 -G videostreamingplatform videostreamingplatform

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/cdn-invalidator .

# Change ownership to non-root user
RUN chown -R videostreamingplatform:videostreamingplatform /app

# Switch to non-root user
USER videostreamingplatform

# No health check — this is a Kafka consumer, not an HTTP server
# Liveness is monitored via Kafka consumer group lag

# Run the application
CMD ["./cdn-invalidator"]
