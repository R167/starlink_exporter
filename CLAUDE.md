# Starlink Prometheus Exporter

## Goal
Build a Prometheus exporter for Starlink Dishy metrics in Go.

## API Details
- **Endpoint:** `192.168.100.1:9200` (configurable via `--dish` flag)
- **Protocol:** gRPC with native protobuf support
- **Service:** `SpaceX.API.Device.Device/Handle`
- **Key Methods:**
  - `get_status` - Current metrics (throughput, latency, obstruction, device info)
  - `get_history` - Historical arrays (15 minutes of data, 900 samples @ 1sec intervals)

### Example Commands
```bash
# Get status
grpcurl -plaintext -d '{"get_status":{}}' 192.168.100.1:9200 SpaceX.API.Device.Device/Handle

# Get history
grpcurl -plaintext -d '{"get_history":{}}' 192.168.100.1:9200 SpaceX.API.Device.Device/Handle

# Describe service
grpcurl -plaintext 192.168.100.1:9200 describe SpaceX.API.Device.Device.Handle
```

## Requirements

### Must Have
- **Language:** Go
- **Metrics Port:** 9999 (default, configurable via `--listen`)
- **Export as counters where feasible**
- **Bandwidth counter:** MUST track cumulative upload/download bytes
  - Source: `downlinkThroughputBps` and `uplinkThroughputBps` arrays from `get_history`
  - **CRITICAL:** Arrays are **circular buffers** indexed by `Current % 900`
  - Background ticker updates every 1 second (independent of Prometheus scrapes)
  - Resilient to network issues, concurrent scrapes, and dishy restarts
- **Structured logging:** Uses `log/slog` with configurable levels
- **Follow Go best practices**
- **Follow Prometheus best practices**

### Guidelines
- Err on side of collecting MORE stats (filter later if needed)
- Use `go doc` for documentation lookup
- Install packages with `@latest` version
- Use `go run` for testing (not `go build`)
- Use background shell mode when running servers for testing

## Architecture

### Critical Discovery: Circular Buffer Arrays
The history arrays from `get_history` are **circular buffers**:
- **Array length:** 900 samples (15 minutes @ 1 sample/second)
- **Current field:** Timestamp indicating "now"
- **Index calculation:** `index = Current % 900`
- **New samples:** From `(lastCurrent + 1) % 900` to `Current % 900`

Example:
```
Current = 167382
Current index = 167382 % 900 = 882
Values at indices 882, 883, 884... contain the most recent data
```

### Metrics Tracker Implementation
**Background ticker approach (runs every 1 second):**
1. Track `lastCurrent` timestamp
2. On each tick (1 second later):
   - Calculate `timeDelta = current - lastCurrent`
   - Iterate circular buffer: `for i = 0; i < timeDelta; i++: index = (lastCurrent + 1 + i) % 900`
   - **Bandwidth**: Convert `bytes = throughputBps[index] / 8.0` (bits to bytes)
   - **Energy**: Integrate `joules = powerWatts[index] * 1 second`
   - **Ping latency**: Accumulate `seconds += pingLatencyMs[index] / 1000.0` (convert to SI units)
   - **Ping drops**: Count `drops += pingDropRate[index]`
   - Accumulate all metrics into counters
3. Thread-safe with `sync.RWMutex`
4. Handles counter resets (dishy restarts)
5. Continues operating even when dish is unreachable

### Metrics Exported
- **Counters** (integrated from circular buffer history):
  - `starlink_download_bytes_total` - Cumulative download bytes
  - `starlink_upload_bytes_total` - Cumulative upload bytes
  - `starlink_energy_joules_total` - Cumulative energy consumed (joules = watt-seconds)
  - `starlink_ping_latency_seconds_sum` - Sum of ping latencies in seconds (summary metric)
  - `starlink_ping_latency_seconds_count` - Count of ping samples (summary metric)
  - `starlink_ping_drop_total` - Total ping drops

- **Gauges:**
  - `starlink_downlink_throughput_bps` - Current downlink throughput
  - `starlink_uplink_throughput_bps` - Current uplink throughput
  - `starlink_pop_ping_latency_ms` - Latency to POP
  - `starlink_uptime_seconds` - Device uptime
  - `starlink_obstruction_fraction` - Fraction of time obstructed
  - `starlink_obstruction_valid_seconds` - Valid observation time
  - `starlink_gps_satellites` - Number of GPS satellites
  - `starlink_gps_valid` - GPS validity (1=valid, 0=invalid)
  - `starlink_eth_speed_mbps` - Ethernet speed
  - `starlink_snr_above_noise_floor` - SNR status (1=yes, 0=no)
  - `starlink_up` - Scrape success indicator (1=success, 0=failure)

- **Info:**
  - `starlink_info{id, hardware_version, software_version, country_code}` - Device metadata

### Dependencies
- `google.golang.org/grpc@latest` - gRPC client
- `google.golang.org/protobuf@latest` - Protobuf support
- `github.com/prometheus/client_golang@latest` - Prometheus client
- Generated protobuf files in `proto/spacex_api/device/`

### Proto File Generation
Proto files are **not checked into git** - they're fetched from the Starlink dish via gRPC reflection:

```bash
# Fetch .proto files and generate Go code
make proto

# This runs:
# 1. grpcurl -proto-out-dir to fetch .proto files via reflection
# 2. sed to add go_package options to proto files
# 3. protoc to generate .pb.go files
```

The `.proto` files are in `.gitignore` but the generated `.pb.go` files are committed.

## Project Structure
```
cmd/exporter/main.go          # Entry point with flags, graceful shutdown
internal/
  collector/
    bandwidth.go              # Background bandwidth tracker
    starlink.go               # Prometheus collector
  client/
    native.go                 # Native gRPC client
    starlink.go               # Client interface and types
proto/spacex_api/             # Generated protobuf files
```

## Configuration Flags
```bash
--listen string      # HTTP metrics address (default ":9999")
--dish string        # Starlink dish gRPC address (default "192.168.100.1:9200")
--log-level string   # Log level: debug, info, warn, error (default "info")
```

## Running the Exporter
```bash
# Default configuration
go run ./cmd/exporter

# Custom configuration
go run ./cmd/exporter --listen :8080 --dish 192.168.100.1:9200 --log-level debug

# Test metrics endpoint
curl -s localhost:9999/metrics | grep starlink_

# Monitor bandwidth counters
watch -n 1 'curl -s localhost:9999/metrics | grep _bytes_total'
```

## Testing
```bash
# Run unit tests
go test ./internal/collector/... -v

# Run all tests, vet, and format
go test ./... && go vet ./... && go fmt ./...
```

## Debugging
Enable debug logging to see:
- Prometheus scrape events
- Metrics tracker updates with sample indices (every second)
- Download/upload deltas (bytes per second)
- Energy consumption deltas (joules per second)
- Ping latency deltas (seconds) and drop deltas
- Cumulative totals for all counters

```bash
go run ./cmd/exporter --log-level debug
```

Example debug output:
```
time=2025-10-04T09:14:05.272Z level=DEBUG msg="Metrics update"
  time_delta=1 sample_indices=[529]
  download_delta_bytes=48472.5234375
  upload_delta_bytes=38558.23828125
  energy_delta_joules=48.950096130371094
  ping_latency_delta_seconds=0.022024417877197266
  ping_sample_count=1
  ping_drop_delta=0
```

## Key Implementation Details

### Thread Safety
- `BandwidthTracker` uses `sync.RWMutex`
- Read lock for `GetCounters()` during Prometheus scrapes
- Write lock for bandwidth updates
- `sync.Once` prevents double-close panic on `Stop()`

### Counter Reset Detection
When dishy restarts, `Current` timestamp may reset:
```go
if current < bt.lastCurrent {
    // Log warning but keep accumulating counters
    bt.lastCurrent = current
    return
}
```

### Graceful Shutdown
1. Intercept SIGINT/SIGTERM signals
2. Cancel context (stops bandwidth tracker)
3. Shutdown HTTP server with timeout
4. Wait for bandwidth tracker to stop
5. Close gRPC connection

## Makefile Targets
```bash
make proto         # Fetch proto files via reflection and generate Go code
make fetch-protos  # Fetch .proto files from dish via grpcurl
make fix-protos    # Add go_package options to proto files
make test          # Run tests
make run           # Run the exporter
make dev           # Rebuild protos and run
make clean         # Remove generated files (proto dir)
```

## Available Tools
- `grpcurl` - gRPC client with reflection support
- `jq` - JSON processing
- Pre-approved commands: `go doc`, `go mod`, `go fmt`, `go test`, `go vet`, `go get`, `go run ./cmd/:*`, `grpcurl`, `curl -s localhost:9999/metrics`

## Using the Metrics

### Average Ping Latency
The `starlink_ping_latency_seconds_sum` and `starlink_ping_latency_seconds_count` summary metrics can be used with Prometheus to calculate average ping latency:

```promql
# Average ping latency over last 5 minutes (in seconds)
rate(starlink_ping_latency_seconds_sum[5m]) / rate(starlink_ping_latency_seconds_count[5m])

# Or for instantaneous average latency (in milliseconds for display)
(starlink_ping_latency_seconds_sum / starlink_ping_latency_seconds_count) * 1000
```

### Energy Consumption
The `starlink_energy_joules_total` counter tracks total energy in joules (watt-seconds):
- 1 joule = 1 watt × 1 second
- To get average power in watts: `rate(starlink_energy_joules_total[5m])`
- To get total energy in kWh: `starlink_energy_joules_total / 3600000`

## Lessons Learned

### Critical Bug Fixed: Array Indexing
**Initial (WRONG) assumption:** Arrays are ordered oldest-to-newest, with newest at index 899.

**Reality:** Arrays are **circular buffers**. The `Current` timestamp indicates which index is "now" via modulo: `index = Current % 900`.

**Impact:** Initial implementation read the wrong samples, resulting in constant bandwidth values instead of actual varying throughput.

**Fix:** Implemented proper circular buffer iteration from `(lastCurrent + 1) % 900` to `Current % 900`.
