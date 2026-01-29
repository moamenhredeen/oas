package benchmarker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
	"github.com/moamenhredeen/oas/internal/tester"
	"golang.org/x/time/rate"
)

// EventType represents the type of benchmark event
type EventType int

const (
	// EventWarmupStarting indicates warmup phase is starting for an endpoint
	EventWarmupStarting EventType = iota
	// EventWarmupProgress indicates warmup progress
	EventWarmupProgress
	// EventWarmupCompleted indicates warmup phase completed
	EventWarmupCompleted
	// EventBenchmarkStarting indicates benchmark is starting for an endpoint
	EventBenchmarkStarting
	// EventBenchmarkProgress indicates benchmark progress (periodic updates)
	EventBenchmarkProgress
	// EventBenchmarkCompleted indicates benchmark completed for an endpoint
	EventBenchmarkCompleted
)

// BenchmarkEvent represents an event during benchmark execution
type BenchmarkEvent struct {
	Type      EventType
	Operation models.Operation
	Result    *models.BenchmarkResult // nil until completed
	Index     int                     // current endpoint index (0-based)
	Total     int                     // total number of endpoints
	Progress  int                     // current iteration count
	MaxIter   int                     // max iterations for this phase

	// Running stats (for progress events)
	RunningAvg    time.Duration
	RunningReqSec float64
	ErrorCount    int
}

// OnBenchmarkEvent is a callback function for benchmark events
type OnBenchmarkEvent func(event BenchmarkEvent)

// Config holds benchmark configuration
type Config struct {
	Iterations       int           // Number of requests per endpoint
	Concurrency      int           // Number of concurrent workers
	WarmupRuns       int           // Number of warmup iterations (discarded)
	RateLimit        float64       // Max requests per second (0 = unlimited)
	Timeout          time.Duration // Per-request timeout
	DisableKeepAlive bool          // Disable HTTP connection reuse
}

// DefaultConfig returns default benchmark configuration
func DefaultConfig() Config {
	return Config{
		Iterations:       100,
		Concurrency:      1,
		WarmupRuns:       5,
		RateLimit:        0,
		Timeout:          30 * time.Second,
		DisableKeepAlive: false,
	}
}

// Benchmarker executes API benchmarks based on OpenAPI specifications
type Benchmarker struct {
	config         Config
	requestBuilder *tester.RequestBuilder
	client         *http.Client
	limiter        *rate.Limiter
}

// NewBenchmarker creates a new benchmarker instance
func NewBenchmarker(config Config) *Benchmarker {
	// Create HTTP transport with keepalive settings
	transport := &http.Transport{
		DisableKeepAlives:   config.DisableKeepAlive,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: config.Concurrency,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	// Create rate limiter if configured
	var limiter *rate.Limiter
	if config.RateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(config.RateLimit), int(config.RateLimit))
	}

	return &Benchmarker{
		config:         config,
		requestBuilder: tester.NewRequestBuilder(),
		client:         client,
		limiter:        limiter,
	}
}

// requestResult holds the result of a single request
type requestResult struct {
	Duration   time.Duration
	StatusCode int
	Error      string
}

// BenchmarkOperation benchmarks a single API operation
func (b *Benchmarker) BenchmarkOperation(
	ctx context.Context,
	op models.Operation,
	p *parser.Parser,
	onEvent OnBenchmarkEvent,
	index, total int,
) (models.BenchmarkResult, error) {
	result := models.BenchmarkResult{
		Path:        op.Path,
		Method:      op.Method,
		OperationID: op.OperationID,
		Iterations:  b.config.Iterations,
		Concurrency: b.config.Concurrency,
		WarmupRuns:  b.config.WarmupRuns,
		StatusCodes: make(map[int]int),
	}

	// Get operation details for request building
	opDetails, err := p.GetOperationDetails(op.Path, op.Method)
	if err != nil {
		return result, fmt.Errorf("failed to get operation details: %w", err)
	}

	// Build a sample request to validate
	_, err = b.requestBuilder.BuildRequest(opDetails, op.ServerURL)
	if err != nil {
		return result, fmt.Errorf("failed to build request: %w", err)
	}

	// Warmup phase
	if b.config.WarmupRuns > 0 && onEvent != nil {
		onEvent(BenchmarkEvent{
			Type:      EventWarmupStarting,
			Operation: op,
			Index:     index,
			Total:     total,
			MaxIter:   b.config.WarmupRuns,
		})
	}

	// Run warmup (single-threaded, no stats collection)
	for i := 0; i < b.config.WarmupRuns; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		b.executeRequest(ctx, opDetails, op.ServerURL)

		if onEvent != nil && (i+1)%max(1, b.config.WarmupRuns/5) == 0 {
			onEvent(BenchmarkEvent{
				Type:      EventWarmupProgress,
				Operation: op,
				Index:     index,
				Total:     total,
				Progress:  i + 1,
				MaxIter:   b.config.WarmupRuns,
			})
		}
	}

	if b.config.WarmupRuns > 0 && onEvent != nil {
		onEvent(BenchmarkEvent{
			Type:      EventWarmupCompleted,
			Operation: op,
			Index:     index,
			Total:     total,
		})
	}

	// Benchmark phase
	if onEvent != nil {
		onEvent(BenchmarkEvent{
			Type:      EventBenchmarkStarting,
			Operation: op,
			Index:     index,
			Total:     total,
			MaxIter:   b.config.Iterations,
		})
	}

	// Execute benchmark with concurrency
	startTime := time.Now()
	results := b.runConcurrentBenchmark(ctx, opDetails, op.ServerURL, onEvent, op, index, total)
	result.TotalDuration = time.Since(startTime)

	// Process results
	result = b.processResults(result, results)

	if onEvent != nil {
		onEvent(BenchmarkEvent{
			Type:      EventBenchmarkCompleted,
			Operation: op,
			Result:    &result,
			Index:     index,
			Total:     total,
		})
	}

	return result, nil
}

// runConcurrentBenchmark executes the benchmark with worker pool
func (b *Benchmarker) runConcurrentBenchmark(
	ctx context.Context,
	opDetails *parser.OperationDetails,
	serverURL string,
	onEvent OnBenchmarkEvent,
	op models.Operation,
	index, total int,
) []requestResult {
	results := make([]requestResult, b.config.Iterations)
	jobs := make(chan int, b.config.Iterations)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var completed int
	var totalDuration time.Duration
	var errorCount int

	// Progress reporting interval
	progressInterval := max(1, b.config.Iterations/20) // ~5% intervals

	// Start workers
	for w := 0; w < b.config.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Apply rate limiting
				if b.limiter != nil {
					b.limiter.Wait(ctx)
				}

				res := b.executeRequest(ctx, opDetails, serverURL)
				results[i] = res

				// Update progress
				mu.Lock()
				completed++
				totalDuration += res.Duration
				if res.Error != "" {
					errorCount++
				}
				currentCompleted := completed
				currentTotalDuration := totalDuration
				currentErrorCount := errorCount
				mu.Unlock()

				// Report progress periodically
				if onEvent != nil && currentCompleted%progressInterval == 0 {
					avgDuration := currentTotalDuration / time.Duration(currentCompleted)
					elapsed := time.Since(time.Now().Add(-currentTotalDuration))
					reqsPerSec := float64(currentCompleted) / elapsed.Seconds()
					if elapsed.Seconds() == 0 {
						reqsPerSec = 0
					}

					onEvent(BenchmarkEvent{
						Type:          EventBenchmarkProgress,
						Operation:     op,
						Index:         index,
						Total:         total,
						Progress:      currentCompleted,
						MaxIter:       b.config.Iterations,
						RunningAvg:    avgDuration,
						RunningReqSec: reqsPerSec,
						ErrorCount:    currentErrorCount,
					})
				}
			}
		}()
	}

	// Send jobs
	for i := 0; i < b.config.Iterations; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}

// executeRequest executes a single HTTP request and returns timing
func (b *Benchmarker) executeRequest(
	ctx context.Context,
	opDetails *parser.OperationDetails,
	serverURL string,
) requestResult {
	result := requestResult{}

	req, err := b.requestBuilder.BuildRequest(opDetails, serverURL)
	if err != nil {
		result.Error = fmt.Sprintf("build request failed: %v", err)
		return result
	}

	req = req.WithContext(ctx)

	startTime := time.Now()
	resp, err := b.client.Do(req)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	return result
}

// processResults calculates statistics from raw results
func (b *Benchmarker) processResults(result models.BenchmarkResult, rawResults []requestResult) models.BenchmarkResult {
	if len(rawResults) == 0 {
		return result
	}

	var durations []time.Duration
	var totalDuration time.Duration
	errorSet := make(map[string]bool)

	for _, r := range rawResults {
		if r.Error != "" {
			result.ErrorCount++
			if len(result.SampleErrors) < 5 && !errorSet[r.Error] {
				result.SampleErrors = append(result.SampleErrors, r.Error)
				errorSet[r.Error] = true
			}
		} else {
			result.SuccessCount++
			durations = append(durations, r.Duration)
			totalDuration += r.Duration
		}

		if r.StatusCode > 0 {
			result.StatusCodes[r.StatusCode]++
		}
	}

	// Calculate timing stats (only from successful requests)
	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		result.MinTime = durations[0]
		result.MaxTime = durations[len(durations)-1]
		result.AvgTime = totalDuration / time.Duration(len(durations))
		result.P50Time = percentile(durations, 50)
		result.P90Time = percentile(durations, 90)
		result.P99Time = percentile(durations, 99)
	}

	// Calculate throughput
	if result.TotalDuration > 0 {
		result.RequestsPerSec = float64(result.Iterations) / result.TotalDuration.Seconds()
	}

	// Calculate error rate
	if result.Iterations > 0 {
		result.ErrorRate = float64(result.ErrorCount) / float64(result.Iterations) * 100
	}

	return result
}

// percentile calculates the p-th percentile from sorted durations
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	index := float64(len(sorted)-1) * float64(p) / 100.0
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// BenchmarkOperations benchmarks multiple operations with live event reporting
func (b *Benchmarker) BenchmarkOperations(
	ctx context.Context,
	operations []models.Operation,
	p *parser.Parser,
	onEvent OnBenchmarkEvent,
) models.BenchmarkSummary {
	summary := models.BenchmarkSummary{
		Iterations:  b.config.Iterations,
		Concurrency: b.config.Concurrency,
		WarmupRuns:  b.config.WarmupRuns,
		Results:     make([]models.BenchmarkResult, 0, len(operations)),
	}

	startTime := time.Now()

	for i, op := range operations {
		select {
		case <-ctx.Done():
			break
		default:
		}

		result, err := b.BenchmarkOperation(ctx, op, p, onEvent, i, len(operations))
		if err != nil {
			result.SampleErrors = append(result.SampleErrors, err.Error())
			result.ErrorCount = result.Iterations
			result.ErrorRate = 100
		}
		summary.AddResult(result)
	}

	summary.Finalize(time.Since(startTime))
	return summary
}
