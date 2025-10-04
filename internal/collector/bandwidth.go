package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/R167/starlink_exporter/internal/client"
)

// BandwidthTracker tracks cumulative metrics from history with a background ticker
// Despite the name, it tracks bandwidth, power, and ping metrics
type BandwidthTracker struct {
	mu                     sync.RWMutex
	client                 client.Client
	logger                 *slog.Logger
	lastCurrent            uint64  // Last seen history timestamp
	downloadBytesTotal     float64 // Cumulative download bytes
	uploadBytesTotal       float64 // Cumulative upload bytes
	energyJoulesTotal      float64 // Cumulative energy consumed (joules = watt-seconds)
	pingLatencySecondsSum  float64 // Sum of ping latencies in seconds (summary metric)
	pingLatencySampleCount float64 // Count of ping samples (summary metric)
	pingDropCount          float64 // Count of ping drops
	lastError              error   // Last error encountered
	initialized            bool
	stopCh                 chan struct{}
	stoppedCh              chan struct{}
	stopOnce               sync.Once
}

// NewBandwidthTracker creates a new bandwidth tracker
func NewBandwidthTracker(client client.Client, logger *slog.Logger) *BandwidthTracker {
	return &BandwidthTracker{
		client:    client,
		logger:    logger,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// Start begins the background ticker that updates bandwidth counters every second
func (bt *BandwidthTracker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer close(bt.stoppedCh)

	bt.logger.Info("Bandwidth tracker started")

	for {
		select {
		case <-ctx.Done():
			bt.logger.Info("Bandwidth tracker stopping")
			return
		case <-bt.stopCh:
			bt.logger.Info("Bandwidth tracker stopping")
			return
		case <-ticker.C:
			bt.update()
		}
	}
}

// Stop stops the bandwidth tracker (safe to call multiple times)
func (bt *BandwidthTracker) Stop() {
	bt.stopOnce.Do(func() {
		close(bt.stopCh)
	})
	<-bt.stoppedCh
}

// update fetches history and updates counters (called every second by ticker)
func (bt *BandwidthTracker) update() {
	history, err := bt.client.GetHistory()
	if err != nil {
		bt.mu.Lock()
		bt.lastError = err
		bt.mu.Unlock()
		bt.logger.Warn("Failed to get history", "error", err)
		return
	}

	bt.processHistory(history)
}

// processHistory processes new history data and updates counters
func (bt *BandwidthTracker) processHistory(history *client.HistoryResponse) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Clear error on successful fetch
	bt.lastError = nil

	// Validate array lengths match
	arrayLen := len(history.DownlinkThroughputBps)
	if arrayLen == 0 {
		bt.logger.Warn("Empty history arrays")
		return
	}

	if len(history.UplinkThroughputBps) != arrayLen ||
		len(history.PowerIn) != arrayLen ||
		len(history.PopPingLatencyMs) != arrayLen ||
		len(history.PopPingDropRate) != arrayLen {
		bt.logger.Error("Array length mismatch",
			"downlink", arrayLen,
			"uplink", len(history.UplinkThroughputBps),
			"power", len(history.PowerIn),
			"ping_latency", len(history.PopPingLatencyMs),
			"ping_drop", len(history.PopPingDropRate))
		return
	}

	// Parse current timestamp - now using uint64 directly
	current := history.Current

	// On first run, just record the current timestamp
	if !bt.initialized {
		bt.lastCurrent = current
		bt.initialized = true
		bt.logger.Info("Bandwidth tracker initialized", "current", current)
		return
	}

	// Detect counter reset (dishy restart)
	if current < bt.lastCurrent {
		bt.logger.Warn("Counter reset detected (dishy restart?)",
			"previous", bt.lastCurrent,
			"current", current)
		bt.lastCurrent = current
		// Don't reset counters - keep accumulating across restarts
		return
	}

	// Calculate how many new samples we have
	timeDelta := current - bt.lastCurrent

	if timeDelta == 0 {
		// No new data yet
		return
	}

	// The history arrays are CIRCULAR BUFFERS
	// Current timestamp tells us which index is "now": index = Current % arrayLength
	bufferLen := uint64(arrayLen)

	// Cap timeDelta to avoid processing more than the buffer size
	if timeDelta > bufferLen {
		bt.logger.Warn("Time delta exceeds history buffer size, possible data loss",
			"delta", timeDelta,
			"buffer_size", bufferLen)
		timeDelta = bufferLen
	}

	// Iterate through the circular buffer from lastCurrent+1 to lastCurrent+timeDelta
	// Integrate all metrics: bandwidth (bytes), power (joules), ping latency (ms), ping drops (count)
	var downloadDelta, uploadDelta, energyDelta, pingLatencyDelta, pingDropDelta float64
	var sampleIndices []int

	for i := uint64(0); i < timeDelta; i++ {
		t := bt.lastCurrent + i
		idx := int(t % bufferLen)

		// Bandwidth: convert bits/sec to bytes (each sample = 1 second)
		downloadDelta += history.DownlinkThroughputBps[idx] / 8.0
		uploadDelta += history.UplinkThroughputBps[idx] / 8.0

		// Power: watts * 1 second = joules
		energyDelta += history.PowerIn[idx]

		// Ping metrics: accumulate latency (convert ms to seconds) and drops
		pingLatencyDelta += history.PopPingLatencyMs[idx] / 1000.0 // ms to seconds
		pingDropDelta += history.PopPingDropRate[idx]

		// Log first few sample indices for debugging
		if len(sampleIndices) < 3 {
			sampleIndices = append(sampleIndices, idx)
		}
	}

	bt.downloadBytesTotal += downloadDelta
	bt.uploadBytesTotal += uploadDelta
	bt.energyJoulesTotal += energyDelta
	bt.pingLatencySecondsSum += pingLatencyDelta
	bt.pingLatencySampleCount += float64(timeDelta) // Each sample counted
	bt.pingDropCount += pingDropDelta

	bt.logger.Debug("Metrics update",
		"time_delta", timeDelta,
		"sample_indices", sampleIndices,
		"download_delta_bytes", downloadDelta,
		"upload_delta_bytes", uploadDelta,
		"energy_delta_joules", energyDelta,
		"ping_latency_delta_seconds", pingLatencyDelta,
		"ping_sample_count", timeDelta,
		"ping_drop_delta", pingDropDelta)

	bt.lastCurrent = current
}

// GetCounters returns current bandwidth counters (thread-safe for Prometheus scrapes)
func (bt *BandwidthTracker) GetCounters() (download, upload float64) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.downloadBytesTotal, bt.uploadBytesTotal
}

// GetEnergyJoules returns cumulative energy consumed in joules
func (bt *BandwidthTracker) GetEnergyJoules() float64 {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.energyJoulesTotal
}

// GetPingMetrics returns ping summary metrics: latency sum (seconds), sample count, drop count
func (bt *BandwidthTracker) GetPingMetrics() (latencySum, sampleCount, dropCount float64) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.pingLatencySecondsSum, bt.pingLatencySampleCount, bt.pingDropCount
}

// GetLastError returns the last error encountered (or nil if no error)
func (bt *BandwidthTracker) GetLastError() error {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.lastError
}
