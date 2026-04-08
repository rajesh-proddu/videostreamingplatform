package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(ctx context.Context) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	bucket := "videostreamingplatform"

	return &S3Client{
		client: client,
		bucket: bucket,
	}, nil
}

// Upload uploads a file chunk to S3
func (s *S3Client) Upload(ctx context.Context, key string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          body,
		ContentLength: &size,
	})
	return err
}

// Download retrieves a file from S3
func (s *S3Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

// Delete removes an object from S3
func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

// Exists checks if an object exists
func (s *S3Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}
