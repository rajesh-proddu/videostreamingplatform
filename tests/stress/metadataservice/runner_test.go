package main

import (
	"testing"
	"time"
)

func TestOperationProfileValidate(t *testing.T) {
	t.Parallel()

	valid := operationProfile{
		ReadByIDPct: 60,
		ListPct:     20,
		CreatePct:   15,
		UpdatePct:   5,
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("expected valid profile, got error: %v", err)
	}

	invalid := operationProfile{
		ReadByIDPct: 50,
		ListPct:     20,
		CreatePct:   15,
		UpdatePct:   5,
	}
	if err := invalid.validate(); err == nil {
		t.Fatal("expected validation error for percentages that do not sum to 100")
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	samples := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	if got := percentile(samples, 50); got != 30*time.Millisecond {
		t.Fatalf("p50 = %s, want %s", got, 30*time.Millisecond)
	}
	if got := percentile(samples, 95); got != 50*time.Millisecond {
		t.Fatalf("p95 = %s, want %s", got, 50*time.Millisecond)
	}
	if got := percentile(nil, 95); got != 0 {
		t.Fatalf("empty percentile = %s, want 0", got)
	}
}
