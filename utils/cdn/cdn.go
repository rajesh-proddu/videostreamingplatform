// Package cdn provides CDN cache invalidation for video content.
package cdn

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// Invalidator defines the interface for CDN cache invalidation.
type Invalidator interface {
	// InvalidateVideo removes a video from the CDN edge cache.
	InvalidateVideo(ctx context.Context, videoID string) error
}

// CloudFrontInvalidator invalidates CloudFront cache entries.
type CloudFrontInvalidator struct {
	client         *cloudfront.Client
	distributionID string
	logger         *log.Logger
}

// NewCloudFrontInvalidator creates a CloudFront invalidator.
// Returns nil if distributionID is empty (CDN disabled).
func NewCloudFrontInvalidator(ctx context.Context, distributionID string, logger *log.Logger) (Invalidator, error) {
	if distributionID == "" {
		return &NoOpInvalidator{}, nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &CloudFrontInvalidator{
		client:         cloudfront.NewFromConfig(cfg),
		distributionID: distributionID,
		logger:         logger,
	}, nil
}

// InvalidateVideo creates a CloudFront invalidation for the video's S3 path.
func (c *CloudFrontInvalidator) InvalidateVideo(ctx context.Context, videoID string) error {
	path := "/videos/" + videoID
	callerRef := fmt.Sprintf("video-delete-%s-%d", videoID, time.Now().UnixMilli())

	_, err := c.client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(c.distributionID),
		InvalidationBatch: &types.InvalidationBatch{
			CallerReference: aws.String(callerRef),
			Paths: &types.Paths{
				Quantity: aws.Int32(1),
				Items:    []string{path},
			},
		},
	})
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("WARN: CloudFront invalidation failed for %s: %v", path, err)
		}
		return fmt.Errorf("cloudfront invalidation failed for %s: %w", path, err)
	}

	if c.logger != nil {
		c.logger.Printf("CloudFront invalidation created for %s", path)
	}
	return nil
}

// NoOpInvalidator does nothing (used when CDN is not configured).
type NoOpInvalidator struct{}

// InvalidateVideo is a no-op.
func (n *NoOpInvalidator) InvalidateVideo(_ context.Context, _ string) error {
	return nil
}
