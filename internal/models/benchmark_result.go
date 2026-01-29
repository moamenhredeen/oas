package models

import "time"

// BenchmarkResult represents the benchmark results for a single API endpoint
type BenchmarkResult struct {
	// Operation details
	Path        string `json:"path"`
	Method      string `json:"method"`
	OperationID string `json:"operation_id,omitempty"`

	// Benchmark configuration
	Iterations  int `json:"iterations"`
	Concurrency int `json:"concurrency"`
	WarmupRuns  int `json:"warmup_runs"`

	// Timing statistics (in nanoseconds for JSON, display as milliseconds)
	MinTime time.Duration `json:"min_time_ns"`
	MaxTime time.Duration `json:"max_time_ns"`
	AvgTime time.Duration `json:"avg_time_ns"`
	P50Time time.Duration `json:"p50_time_ns"`
	P90Time time.Duration `json:"p90_time_ns"`
	P99Time time.Duration `json:"p99_time_ns"`

	// Throughput
	RequestsPerSec float64       `json:"requests_per_sec"`
	TotalDuration  time.Duration `json:"total_duration_ns"`

	// Error tracking
	SuccessCount int     `json:"success_count"`
	ErrorCount   int     `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`

	// Status code distribution
	StatusCodes map[int]int `json:"status_codes"`

	// Sample errors (first few unique errors)
	SampleErrors []string `json:"sample_errors,omitempty"`
}

// BenchmarkSummary represents the overall benchmark results
type BenchmarkSummary struct {
	// Configuration
	TotalEndpoints int `json:"total_endpoints"`
	Iterations     int `json:"iterations_per_endpoint"`
	Concurrency    int `json:"concurrency"`
	WarmupRuns     int `json:"warmup_runs"`

	// Aggregate timing
	OverallMinTime time.Duration `json:"overall_min_time_ns"`
	OverallMaxTime time.Duration `json:"overall_max_time_ns"`
	OverallAvgTime time.Duration `json:"overall_avg_time_ns"`

	// Aggregate throughput
	TotalRequests     int           `json:"total_requests"`
	TotalSuccesses    int           `json:"total_successes"`
	TotalErrors       int           `json:"total_errors"`
	OverallErrorRate  float64       `json:"overall_error_rate"`
	TotalDuration     time.Duration `json:"total_duration_ns"`
	OverallReqsPerSec float64       `json:"overall_requests_per_sec"`

	// Per-endpoint results
	Results []BenchmarkResult `json:"results"`
}

// AddResult adds a benchmark result to the summary and updates aggregates
func (s *BenchmarkSummary) AddResult(result BenchmarkResult) {
	s.Results = append(s.Results, result)
	s.TotalEndpoints = len(s.Results)
	s.TotalRequests += result.Iterations
	s.TotalSuccesses += result.SuccessCount
	s.TotalErrors += result.ErrorCount

	// Update min/max
	if s.OverallMinTime == 0 || result.MinTime < s.OverallMinTime {
		s.OverallMinTime = result.MinTime
	}
	if result.MaxTime > s.OverallMaxTime {
		s.OverallMaxTime = result.MaxTime
	}

	// Recalculate averages and rates
	if s.TotalRequests > 0 {
		s.OverallErrorRate = float64(s.TotalErrors) / float64(s.TotalRequests) * 100
	}

	// Calculate overall average from individual averages (weighted)
	var totalWeightedAvg time.Duration
	var totalWeight int
	for _, r := range s.Results {
		totalWeightedAvg += r.AvgTime * time.Duration(r.Iterations)
		totalWeight += r.Iterations
	}
	if totalWeight > 0 {
		s.OverallAvgTime = totalWeightedAvg / time.Duration(totalWeight)
	}
}

// Finalize calculates final aggregate metrics
func (s *BenchmarkSummary) Finalize(totalDuration time.Duration) {
	s.TotalDuration = totalDuration
	if totalDuration > 0 {
		s.OverallReqsPerSec = float64(s.TotalRequests) / totalDuration.Seconds()
	}
}
