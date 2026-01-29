# OAS - OpenAPI Specification Testing Tool

A command-line tool for testing and benchmarking REST APIs based on OpenAPI Specification files.

## Features

- **API Testing**: Automatically test all endpoints defined in your OpenAPI spec
- **Benchmarking**: Measure API performance with detailed latency metrics
- **Live Output**: Real-time progress reporting with colorful terminal output
- **Filtering**: Test specific endpoints by path, operation ID, or tags
- **Export Results**: Output results in JSON or CSV format
- **Concurrent Requests**: Run parallel requests for load testing
- **Rate Limiting**: Control request rate to avoid overwhelming servers

## Installation

```bash
go install github.com/moamenhredeen/oas@latest
```

Or build from source:

```bash
git clone https://github.com/moamenhredeen/oas.git
cd oas
go build -o oas .
```

## Commands

### test

Test API endpoints defined in an OpenAPI specification file.

```bash
oas test [openapi-spec-file] [flags]
```

**Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--server` | | Override server URL from OpenAPI spec | (from spec) |
| `--filter` | | Filter endpoints by path pattern or operation ID | |
| `--tags` | | Filter by OpenAPI tags (can be repeated) | |
| `--verbose` | `-v` | Show detailed output | `false` |
| `--timeout` | `-t` | Request timeout in seconds | `30` |
| `--output` | `-o` | Output format: `json`, `csv` | |
| `--output-file` | | Write output to file (default: stdout) | |

**Examples:**

```bash
# Test all endpoints
oas test api-spec.json

# Test with custom server URL
oas test api-spec.json --server http://localhost:8080

# Filter by path pattern
oas test api-spec.json --filter /users

# Filter by tags
oas test api-spec.json --tags users --tags admin

# Verbose output with timeout
oas test api-spec.json -v -t 60

# Export results to JSON
oas test api-spec.json -o json --output-file results.json

# Export results to CSV
oas test api-spec.json -o csv --output-file results.csv
```

### benchmark

Benchmark API performance by running multiple iterations of each request and collecting detailed metrics.

```bash
oas benchmark [openapi-spec-file] [flags]
```

**Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--server` | | Override server URL from OpenAPI spec | (from spec) |
| `--filter` | | Filter endpoints by path pattern or operation ID | |
| `--tags` | | Filter by OpenAPI tags | |
| `--verbose` | `-v` | Show detailed output | `false` |
| `--iterations` | `-n` | Number of requests per endpoint | `100` |
| `--concurrency` | `-c` | Number of concurrent requests | `1` |
| `--warmup` | `-w` | Warmup iterations (discarded from stats) | `5` |
| `--rate` | `-r` | Max requests per second (0 = unlimited) | `0` |
| `--timeout` | `-t` | Request timeout in seconds | `30` |
| `--no-keepalive` | | Disable HTTP connection reuse | `false` |
| `--output` | `-o` | Output format: `json`, `csv` | |
| `--output-file` | | Write output to file (default: stdout) | |

**Examples:**

```bash
# Basic benchmark with defaults (100 iterations, 1 concurrent)
oas benchmark api-spec.json

# High-load benchmark with concurrency
oas benchmark api-spec.json -n 1000 -c 10

# Rate-limited benchmark (50 requests per second)
oas benchmark api-spec.json -n 500 --rate 50

# Benchmark with warmup and verbose output
oas benchmark api-spec.json -n 200 -w 10 -v

# Test cold connection performance
oas benchmark api-spec.json --no-keepalive

# Export benchmark results to JSON
oas benchmark api-spec.json -o json --output-file benchmark.json
```

## Output Formats

### Console Output

Both commands provide colorful, real-time console output:

- **Test command**: Shows pass/fail status for each endpoint
- **Benchmark command**: Shows progress, running averages, and final statistics

### JSON Export

Structured JSON output suitable for programmatic processing:

```json
{
  "total_tests": 5,
  "passed": 4,
  "failed": 1,
  "results": [
    {
      "path": "/users",
      "method": "GET",
      "passed": true,
      "status_code": 200,
      "response_time_ns": 45000000
    }
  ]
}
```

### CSV Export

Tabular format suitable for spreadsheets and data analysis:

```csv
method,path,operation_id,passed,status_code,response_time_ms,error
GET,/users,listUsers,true,200,45.00,
POST,/users,createUser,true,201,120.50,
```

## Benchmark Metrics

The benchmark command collects the following metrics:

| Metric | Description |
|--------|-------------|
| **Min** | Minimum response time |
| **Max** | Maximum response time |
| **Avg** | Average response time |
| **P50** | 50th percentile (median) |
| **P90** | 90th percentile |
| **P99** | 99th percentile |
| **Requests/sec** | Throughput |
| **Error Rate** | Percentage of failed requests |
| **Status Codes** | Distribution of HTTP status codes |

## Configuration

OAS supports configuration via a `config.toml` file in the current directory:

```toml
# config.toml example
# (Configuration options TBD)
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All tests passed / benchmark completed |
| `1` | One or more tests failed / error occurred |

## License

See [LICENSE](LICENSE) file.
