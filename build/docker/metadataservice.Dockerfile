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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o metadataservice ./metadataservice

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 videostreamingplatform && \
    adduser -D -u 1000 -G videostreamingplatform videostreamingplatform

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/metadataservice .

# Change ownership to non-root user
RUN chown -R videostreamingplatform:videostreamingplatform /app

# Switch to non-root user
USER videostreamingplatform

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run the application
CMD ["./metadataservice"]
