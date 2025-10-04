# Starlink Prometheus Exporter

A Prometheus exporter for Starlink Dishy metrics written in Go. Exports bandwidth, energy, ping latency, and device statistics from your Starlink terminal.

## Features

- **Bandwidth tracking**: Cumulative upload/download bytes from historical data
- **Energy monitoring**: Total energy consumption in joules
- **Ping metrics**: Latency and drop rate statistics with Prometheus summary metrics
- **Device info**: Hardware version, software version, uptime, GPS status
- **Background ticker**: 1-second updates independent of Prometheus scrapes
- **Resilient**: Handles network issues, concurrent scrapes, and dishy restarts
- **Structured logging**: Configurable log levels with `log/slog`

## Quick Start

```bash
# Run with defaults (listens on :9999, connects to 192.168.100.1:9200)
go run ./cmd/exporter

# Custom configuration
go run ./cmd/exporter --listen :8080 --dish 192.168.100.1:9200 --log-level debug

# View metrics
curl -s localhost:9999/metrics | grep starlink_
```

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd starlink_exporter

# Install dependencies
go mod download

# Generate proto files from Starlink dish (requires grpcurl and protoc)
make proto

# Run tests
go test ./...

# Run the exporter
go run ./cmd/exporter
```

### Generating Proto Files

The proto files are sourced from the Starlink dish via gRPC reflection and are not checked into git. To generate them:

```bash
# Fetch proto files and generate Go code (requires Starlink dish access)
make proto

# Or run each step individually:
make fetch-protos    # Fetch .proto files via grpcurl reflection
make fix-protos      # Add go_package options to proto files
```

**Requirements:**
- `grpcurl` - gRPC client with reflection support
- `protoc` - Protocol buffer compiler
- Access to Starlink dish at `192.168.100.1:9200` (or set `DISH_ADDR` env var)

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `:9999` | HTTP metrics server address |
| `--dish` | `192.168.100.1:9200` | Starlink dish gRPC address |
| `--log-level` | `info` | Log level: debug, info, warn, error |

## Metrics

### Counters (Integrated from Historical Data)
- `starlink_download_bytes_total` - Cumulative download bytes
- `starlink_upload_bytes_total` - Cumulative upload bytes
- `starlink_energy_joules_total` - Total energy consumed (joules)
- `starlink_ping_latency_seconds_sum` - Sum of ping latencies (seconds)
- `starlink_ping_latency_seconds_count` - Count of ping samples
- `starlink_ping_drop_total` - Total ping drops

### Gauges (Current Values)
- `starlink_downlink_throughput_bps` - Current downlink throughput
- `starlink_uplink_throughput_bps` - Current uplink throughput
- `starlink_pop_ping_latency_ms` - Current latency to POP
- `starlink_uptime_seconds` - Device uptime
- `starlink_obstruction_fraction` - Fraction of time obstructed
- `starlink_gps_satellites` - Number of GPS satellites
- `starlink_eth_speed_mbps` - Ethernet speed
- `starlink_up` - Scrape success indicator (1=success, 0=failure)

### Info Labels
- `starlink_info{id, hardware_version, software_version, country_code}` - Device metadata

## Prometheus Queries

### Average Ping Latency (5-minute window)
```promql
rate(starlink_ping_latency_seconds_sum[5m]) / rate(starlink_ping_latency_seconds_count[5m])
```

### Average Power Consumption (watts)
```promql
rate(starlink_energy_joules_total[5m])
```

### Total Energy (kWh)
```promql
starlink_energy_joules_total / 3600000
```

### Bandwidth Rate (bytes/sec)
```promql
rate(starlink_download_bytes_total[5m])
rate(starlink_upload_bytes_total[5m])
```

## Architecture

The exporter uses a **background ticker** that runs every 1 second to:
1. Fetch 15 minutes of historical data from the Starlink dish
2. Process the **circular buffer arrays** (900 samples @ 1sec intervals)
3. Integrate metrics: bandwidth (bits→bytes), energy (watts→joules), ping latency (ms→seconds)
4. Accumulate into thread-safe counters
5. Export to Prometheus on `/metrics`

### Critical: Circular Buffer Arrays
History arrays are **circular buffers** indexed by `Current % 900`:
- Array length: 900 samples (15 minutes)
- Current field: Timestamp indicating "now"
- New samples: From `(lastCurrent + 1) % 900` to `Current % 900`

## API Details

The exporter connects to the Starlink dish gRPC API:
- **Endpoint**: `192.168.100.1:9200` (default)
- **Protocol**: gRPC with native protobuf
- **Service**: `SpaceX.API.Device.Device/Handle`
- **Methods**: `get_status`, `get_history`

## Development

```bash
# Run tests
go test ./internal/collector/... -v

# Run all tests, vet, and format
go test ./... && go vet ./... && go fmt ./...

# Debug logging
go run ./cmd/exporter --log-level debug

# Makefile targets
make proto      # Fetch and generate proto files
make test       # Run tests
make run        # Run the exporter
make dev        # Rebuild protos and run
make clean      # Remove generated files
```

### Debug Output
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

## Dependencies

- `google.golang.org/grpc` - gRPC client
- `google.golang.org/protobuf` - Protobuf support
- `github.com/prometheus/client_golang` - Prometheus client

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details

## Contributing

[Add contribution guidelines here]
