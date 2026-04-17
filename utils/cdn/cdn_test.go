package cdn

import (
	"context"
	"testing"
)

func TestNoOpInvalidator_ReturnsNil(t *testing.T) {
	t.Parallel()
	inv := &NoOpInvalidator{}
	if err := inv.InvalidateVideo(context.Background(), "video-123"); err != nil {
		t.Errorf("NoOpInvalidator should return nil, got %v", err)
	}
}

func TestNewCloudFrontInvalidator_EmptyDistribution_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	inv, err := NewCloudFrontInvalidator(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := inv.(*NoOpInvalidator); !ok {
		t.Error("empty distribution ID should return NoOpInvalidator")
	}
}

func TestNoOpInvalidator_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ Invalidator = &NoOpInvalidator{}
}
