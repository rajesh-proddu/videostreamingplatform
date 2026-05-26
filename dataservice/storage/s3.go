package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrObjectNotFound is returned when a Download targets a key that does not
// exist in the bucket. Handlers map this to 404.
var ErrObjectNotFound = errors.New("object not found")

type S3Client struct {
	client *s3.Client
	bucket string
}

// Object is the result of a Download.
// ContentRange is non-empty when the response is a partial (HTTP 206) result.
type Object struct {
	Body          io.ReadCloser
	ContentLength int64
	ContentType   string
	ContentRange  string
	ETag          string
}

func NewS3Client(ctx context.Context) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	var opts []func(*s3.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, opts...)
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "videostreamingplatform"
	}

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

// CreateMultipart opens an S3 multipart upload for key and returns the S3-assigned
// UploadId. Parts are staged inside S3 (not the server) until CompleteMultipart.
func (s *S3Client) CreateMultipart(ctx context.Context, key string) (string, error) {
	out, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UploadId), nil
}

// UploadPart stages a single part of a multipart upload. partNumber is 1-indexed
// (1..10000). S3 retains each part keyed by (key, uploadID, partNumber); re-uploading
// the same partNumber overwrites it, so chunk retries are idempotent.
func (s *S3Client) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, body io.Reader, size int64) error {
	_, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:        &s.bucket,
		Key:           &key,
		UploadId:      &uploadID,
		PartNumber:    aws.Int32(partNumber),
		Body:          body,
		ContentLength: &size,
	})
	return err
}

// CompleteMultipart finalizes a multipart upload. It lists the parts S3 already
// holds (option 2: no client-side ETag tracking) and asks S3 to concatenate them
// server-side into the final object. The concatenation is atomic — the object
// appears whole or not at all.
func (s *S3Client) CompleteMultipart(ctx context.Context, key, uploadID string) error {
	var parts []s3types.CompletedPart
	var marker *string
	for {
		out, err := s.client.ListParts(ctx, &s3.ListPartsInput{
			Bucket:           &s.bucket,
			Key:              &key,
			UploadId:         &uploadID,
			PartNumberMarker: marker,
		})
		if err != nil {
			return err
		}
		// ListParts returns parts in ascending PartNumber order, which is also the
		// order CompleteMultipartUpload requires.
		for _, p := range out.Parts {
			parts = append(parts, s3types.CompletedPart{
				PartNumber: p.PartNumber,
				ETag:       p.ETag,
			})
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		marker = out.NextPartNumberMarker
	}
	if len(parts) == 0 {
		return fmt.Errorf("no uploaded parts found for multipart upload %s", uploadID)
	}

	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          &s.bucket,
		Key:             &key,
		UploadId:        &uploadID,
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: parts},
	})
	return err
}

// Download retrieves a file (or byte range) from S3.
// Pass an empty rangeHeader for a full download; otherwise pass an HTTP
// Range header value (e.g. "bytes=0-1048575") which is forwarded to S3.
// When S3 returns partial content, Object.ContentRange is populated.
func (s *S3Client) Download(ctx context.Context, key, rangeHeader string) (*Object, error) {
	in := &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}
	if rangeHeader != "" {
		in.Range = &rangeHeader
	}
	result, err := s.client.GetObject(ctx, in)
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, fmt.Errorf("%w: %s", ErrObjectNotFound, key)
		}
		return nil, err
	}
	obj := &Object{Body: result.Body}
	if result.ContentLength != nil {
		obj.ContentLength = *result.ContentLength
	}
	if result.ContentType != nil {
		obj.ContentType = *result.ContentType
	}
	if result.ContentRange != nil {
		obj.ContentRange = *result.ContentRange
	}
	if result.ETag != nil {
		obj.ETag = *result.ETag
	}
	return obj, nil
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

// ListObjects lists objects with the given prefix
func (s *S3Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, obj := range result.Contents {
		keys = append(keys, *obj.Key)
	}
	return keys, nil
}
