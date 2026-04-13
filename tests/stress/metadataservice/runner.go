package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type operationKind string

const (
	opReadByID operationKind = "GET /videos/{id}"
	opList     operationKind = "GET /videos"
	opCreate   operationKind = "POST /videos"
	opUpdate   operationKind = "PUT /videos/{id}"
)

type operationResult struct {
	Operation  operationKind
	StatusCode int
	Latency    time.Duration
	Err        error
}

type operationStats struct {
	Count       int
	Success     int
	Latency     []time.Duration
	StatusCodes map[int]int
}

type summary struct {
	StartedAt    time.Time
	CompletedAt  time.Time
	Issued       int
	Completed    int
	Success      int
	StatusCodes  map[int]int
	Errors       int
	Overall      []time.Duration
	ByOperation  map[operationKind]*operationStats
	TargetQPS    int
	Config       config
	SeededVideos int
}

func newSummary(cfg config, seeded int) *summary {
	return &summary{
		StatusCodes:  make(map[int]int),
		ByOperation:  make(map[operationKind]*operationStats),
		TargetQPS:    cfg.TargetQPS,
		Config:       cfg,
		SeededVideos: seeded,
	}
}

func (s *summary) record(result operationResult) {
	s.Completed++
	s.Overall = append(s.Overall, result.Latency)

	if result.StatusCode != 0 {
		s.StatusCodes[result.StatusCode]++
	}
	if result.Err != nil {
		s.Errors++
	}
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		s.Success++
	}

	stats, ok := s.ByOperation[result.Operation]
	if !ok {
		stats = &operationStats{
			StatusCodes: make(map[int]int),
		}
		s.ByOperation[result.Operation] = stats
	}

	stats.Count++
	stats.Latency = append(stats.Latency, result.Latency)
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		stats.Success++
	}
	if result.StatusCode != 0 {
		stats.StatusCodes[result.StatusCode]++
	}
}

type videoPool struct {
	mu  sync.RWMutex
	ids []string
}

func (p *videoPool) add(id string) {
	if id == "" {
		return
	}
	p.mu.Lock()
	p.ids = append(p.ids, id)
	p.mu.Unlock()
}

func (p *videoPool) random(r *rand.Rand) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.ids) == 0 {
		return "", false
	}
	return p.ids[r.Intn(len(p.ids))], true
}

func (p *videoPool) size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.ids)
}

func (p operationProfile) pick(r *rand.Rand) operationKind {
	value := r.Intn(100)
	switch {
	case value < p.ReadByIDPct:
		return opReadByID
	case value < p.ReadByIDPct+p.ListPct:
		return opList
	case value < p.ReadByIDPct+p.ListPct+p.CreatePct:
		return opCreate
	default:
		return opUpdate
	}
}

func runStress(ctx context.Context, cfg config) error {
	client := newMetadataClient(cfg)
	if err := client.health(ctx); err != nil {
		return fmt.Errorf("metadata service unavailable: %w", err)
	}

	fmt.Printf("Metadata stress test starting\n")
	fmt.Printf("  base_url:        %s\n", cfg.BaseURL)
	fmt.Printf("  target_qps:      %d\n", cfg.TargetQPS)
	fmt.Printf("  duration:        %s\n", cfg.Duration)
	fmt.Printf("  workers:         %d\n", cfg.Workers)
	fmt.Printf("  seed_videos:     %d\n", cfg.SeedVideos)
	fmt.Printf("  request_timeout: %s\n", cfg.RequestTimeout)
	fmt.Printf("  profile:         read-by-id=%d%% list=%d%% create=%d%% update=%d%%\n",
		cfg.Profile.ReadByIDPct, cfg.Profile.ListPct, cfg.Profile.CreatePct, cfg.Profile.UpdatePct)

	pool := &videoPool{}
	seeded, err := seedVideos(ctx, cfg, client, pool)
	if err != nil {
		return err
	}

	results := make(chan operationResult, cfg.Workers*4)
	jobs := make(chan operationKind, cfg.Workers*4)
	done := make(chan struct{})
	summary := newSummary(cfg, seeded)
	summary.StartedAt = time.Now()

	go func() {
		for result := range results {
			summary.record(result)
		}
		close(done)
	}()

	var writeSeq atomic.Uint64
	var workers sync.WaitGroup
	for workerID := 0; workerID < cfg.Workers; workerID++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*7919))
			for operation := range jobs {
				started := time.Now()
				statusCode, err := executeOperation(ctx, cfg, client, pool, operation, cfg.RunTag, writeSeq.Add(1), rng)
				results <- operationResult{
					Operation:  operation,
					StatusCode: statusCode,
					Latency:    time.Since(started),
					Err:        err,
				}
			}
		}(workerID)
	}

	issued := dispatchLoad(ctx, cfg, jobs)
	workers.Wait()
	close(results)
	<-done

	summary.Issued = issued
	summary.CompletedAt = time.Now()
	printSummary(summary)

	return nil
}

func seedVideos(ctx context.Context, cfg config, client *metadataClient, pool *videoPool) (int, error) {
	if cfg.SeedVideos == 0 {
		return 0, nil
	}

	fmt.Printf("Seeding %d metadata rows with %d workers...\n", cfg.SeedVideos, cfg.SeedWorkers)

	jobs := make(chan int)
	var seeded atomic.Int64
	var firstErr atomic.Pointer[error]
	var workers sync.WaitGroup

	for workerID := 0; workerID < min(cfg.SeedWorkers, max(cfg.SeedVideos, 1)); workerID++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				videoID, _, err := client.createVideo(ctx, cfg.RunTag+"-seed", uint64(job+1))
				if err != nil {
					errCopy := err
					firstErr.CompareAndSwap(nil, &errCopy)
					return
				}
				pool.add(videoID)
				seeded.Add(1)
			}
		}(workerID)
	}

	for i := 0; i < cfg.SeedVideos; i++ {
		if firstErr.Load() != nil {
			break
		}
		jobs <- i
	}
	close(jobs)
	workers.Wait()

	if errPtr := firstErr.Load(); errPtr != nil {
		return int(seeded.Load()), fmt.Errorf("seed failed after %d videos: %w", seeded.Load(), *errPtr)
	}

	fmt.Printf("Seed complete: %d metadata rows ready for read/update traffic\n", seeded.Load())
	return int(seeded.Load()), nil
}

func dispatchLoad(ctx context.Context, cfg config, jobs chan<- operationKind) int {
	defer close(jobs)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()
	issued := 0

	for {
		if time.Since(start) >= cfg.Duration {
			return issued
		}

		scheduledAt := start.Add(time.Duration(issued) * time.Second / time.Duration(cfg.TargetQPS))
		if delay := time.Until(scheduledAt); delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return issued
			case <-timer.C:
			}
		}

		select {
		case <-ctx.Done():
			return issued
		case jobs <- cfg.Profile.pick(rng):
			issued++
		}
	}
}

func executeOperation(
	ctx context.Context,
	cfg config,
	client *metadataClient,
	pool *videoPool,
	operation operationKind,
	runTag string,
	seq uint64,
	rng *rand.Rand,
) (int, error) {
	switch operation {
	case opReadByID:
		videoID, ok := pool.random(rng)
		if !ok {
			createdID, status, err := client.createVideo(ctx, runTag, seq)
			if err == nil {
				pool.add(createdID)
			}
			return status, err
		}
		return client.getVideo(ctx, videoID)
	case opList:
		size := pool.size()
		maxOffset := max(size-cfg.ListLimit, 0)
		offset := 0
		if maxOffset > 0 {
			offset = rng.Intn(maxOffset + 1)
		}
		return client.listVideos(ctx, cfg.ListLimit, offset)
	case opCreate:
		videoID, status, err := client.createVideo(ctx, runTag, seq)
		if err == nil {
			pool.add(videoID)
		}
		return status, err
	case opUpdate:
		videoID, ok := pool.random(rng)
		if !ok {
			createdID, status, err := client.createVideo(ctx, runTag, seq)
			if err == nil {
				pool.add(createdID)
			}
			return status, err
		}
		return client.updateVideo(ctx, videoID, runTag, seq)
	default:
		return 0, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func printSummary(s *summary) {
	runDuration := s.CompletedAt.Sub(s.StartedAt)
	achievedQPS := 0.0
	if runDuration > 0 {
		achievedQPS = float64(s.Completed) / runDuration.Seconds()
	}

	fmt.Println("\n=== Stress Test Summary ===")
	fmt.Printf("Target QPS:           %d\n", s.TargetQPS)
	fmt.Printf("Achieved QPS:         %.2f\n", achievedQPS)
	fmt.Printf("Duration:             %s\n", runDuration.Round(time.Millisecond))
	fmt.Printf("Issued requests:      %d\n", s.Issued)
	fmt.Printf("Completed requests:   %d\n", s.Completed)
	fmt.Printf("Successful requests:  %d\n", s.Success)
	fmt.Printf("Errored requests:     %d\n", s.Errors)
	fmt.Printf("Seeded videos:        %d\n", s.SeededVideos)
	fmt.Printf("Latency p50/p95/p99:  %s / %s / %s\n",
		percentile(s.Overall, 50),
		percentile(s.Overall, 95),
		percentile(s.Overall, 99),
	)

	fmt.Println("\nStatus codes:")
	for _, code := range sortedStatusCodes(s.StatusCodes) {
		fmt.Printf("  %d -> %d\n", code, s.StatusCodes[code])
	}

	fmt.Println("\nPer operation:")
	for _, operation := range []operationKind{opReadByID, opList, opCreate, opUpdate} {
		stats, ok := s.ByOperation[operation]
		if !ok {
			continue
		}

		successRate := 0.0
		if stats.Count > 0 {
			successRate = 100 * float64(stats.Success) / float64(stats.Count)
		}

		fmt.Printf(
			"  %-18s count=%-8d success=%-8d rate=%6.2f%% p95=%-10s p99=%-10s\n",
			operation,
			stats.Count,
			stats.Success,
			successRate,
			percentile(stats.Latency, 95),
			percentile(stats.Latency, 99),
		)
	}

	if achievedQPS < float64(s.TargetQPS) {
		gapPct := 100 * (1 - achievedQPS/float64(s.TargetQPS))
		fmt.Printf("\nNote: achieved throughput is %.2f%% below the configured target.\n", math.Max(gapPct, 0))
	}
}

func percentile(samples []time.Duration, p int) time.Duration {
	if len(samples) == 0 {
		return 0
	}

	clone := append([]time.Duration(nil), samples...)
	sort.Slice(clone, func(i, j int) bool { return clone[i] < clone[j] })

	if p <= 0 {
		return clone[0]
	}
	if p >= 100 {
		return clone[len(clone)-1]
	}

	index := int(math.Ceil((float64(p)/100)*float64(len(clone)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(clone) {
		index = len(clone) - 1
	}

	return clone[index]
}

func sortedStatusCodes(counts map[int]int) []int {
	keys := make([]int, 0, len(counts))
	for code := range counts {
		keys = append(keys, code)
	}
	sort.Ints(keys)
	return keys
}

func (s *summary) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("completed=%d success=%d errors=%d", s.Completed, s.Success, s.Errors))
	return builder.String()
}
