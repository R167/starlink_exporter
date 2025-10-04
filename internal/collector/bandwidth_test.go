package collector

import (
	"log/slog"
	"os"
	"testing"

	"github.com/R167/starlink_exporter/internal/client"
)

func TestBandwidthTracker_FirstUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	history := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: []float64{8000, 16000, 24000}, // 3 samples
		UplinkThroughputBps:   []float64{4000, 8000, 12000},
		PowerIn:               []float64{50, 50, 50},
		PopPingLatencyMs:      []float64{20, 20, 20},
		PopPingDropRate:       []float64{0, 0, 0},
	}

	tracker.processHistory(history)

	// First update should just initialize, not accumulate
	download, upload := tracker.GetCounters()
	if download != 0 {
		t.Errorf("Expected 0 download bytes on first update, got %f", download)
	}
	if upload != 0 {
		t.Errorf("Expected 0 upload bytes on first update, got %f", upload)
	}
	if tracker.lastCurrent != 1000 {
		t.Errorf("Expected lastCurrent=1000, got %d", tracker.lastCurrent)
	}
	if !tracker.initialized {
		t.Error("Expected tracker to be initialized")
	}
}

func TestBandwidthTracker_SecondUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	// First update at timestamp 1000
	history1 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history1)

	// Second update at timestamp 1003 (3 seconds later)
	// Circular buffer: new samples at indices 1000%900=100, 1001%900=101, 1002%900=102
	// Index 1003%900=103 is the NEXT sample to be written (not yet valid)
	history2 := &client.HistoryResponse{
		Current:               1003,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	// Set values at circular buffer indices
	history2.DownlinkThroughputBps[100] = 8000  // 1000 bytes/sec
	history2.DownlinkThroughputBps[101] = 16000 // 2000 bytes/sec
	history2.DownlinkThroughputBps[102] = 24000 // 3000 bytes/sec
	history2.UplinkThroughputBps[100] = 4000    // 500 bytes/sec
	history2.UplinkThroughputBps[101] = 8000    // 1000 bytes/sec
	history2.UplinkThroughputBps[102] = 12000   // 1500 bytes/sec

	tracker.processHistory(history2)

	// Expected download: (8000 + 16000 + 24000) / 8 = 6000 bytes
	expectedDownload := 6000.0
	download, upload := tracker.GetCounters()
	if download != expectedDownload {
		t.Errorf("Expected %f download bytes, got %f", expectedDownload, download)
	}

	// Expected upload: (4000 + 8000 + 12000) / 8 = 3000 bytes
	expectedUpload := 3000.0
	if upload != expectedUpload {
		t.Errorf("Expected %f upload bytes, got %f", expectedUpload, upload)
	}
}

func TestBandwidthTracker_CumulativeUpdates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	// First update
	history1 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history1)

	// Second update: 2 seconds later
	// Circular indices: 1000%900=100, 1001%900=101
	history2 := &client.HistoryResponse{
		Current:               1002,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	history2.DownlinkThroughputBps[100] = 8000  // 1000 bytes/sec
	history2.DownlinkThroughputBps[101] = 8000  // 1000 bytes/sec
	history2.UplinkThroughputBps[100] = 0       // 0 bytes/sec
	history2.UplinkThroughputBps[101] = 8000    // 1000 bytes/sec

	tracker.processHistory(history2)

	downloadAfterFirst, uploadAfterFirst := tracker.GetCounters()

	// Third update: 1 second later
	// Circular index: 1002%900=102
	history3 := &client.HistoryResponse{
		Current:               1003,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	history3.DownlinkThroughputBps[102] = 16000 // 2000 bytes/sec
	history3.UplinkThroughputBps[102] = 24000   // 3000 bytes/sec

	tracker.processHistory(history3)

	// Should accumulate
	expectedDownload := downloadAfterFirst + 2000.0
	download, upload := tracker.GetCounters()
	if download != expectedDownload {
		t.Errorf("Expected cumulative download %f, got %f", expectedDownload, download)
	}

	expectedUpload := uploadAfterFirst + 3000.0
	if upload != expectedUpload {
		t.Errorf("Expected cumulative upload %f, got %f", expectedUpload, upload)
	}
}

func TestBandwidthTracker_NoTimeDelta(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	history1 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history1)

	initialDownload, initialUpload := tracker.GetCounters()

	// Same timestamp - should not update
	history2 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history2)

	download, upload := tracker.GetCounters()
	if download != initialDownload {
		t.Errorf("Download bytes should not change with same timestamp")
	}
	if upload != initialUpload {
		t.Errorf("Upload bytes should not change with same timestamp")
	}
}

func TestBandwidthTracker_LargeDelta(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	history1 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history1)

	// Delta larger than array size - should cap at array length
	history2 := &client.HistoryResponse{
		Current:               2000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	// Fill with known values
	for i := range history2.DownlinkThroughputBps {
		history2.DownlinkThroughputBps[i] = 8000 // 1000 bytes/sec
		history2.UplinkThroughputBps[i] = 4000   // 500 bytes/sec
	}

	tracker.processHistory(history2)

	// Should process all 900 samples
	expectedDownload := 900 * 1000.0
	expectedUpload := 900 * 500.0

	download, upload := tracker.GetCounters()
	if download != expectedDownload {
		t.Errorf("Expected %f download bytes, got %f", expectedDownload, download)
	}
	if upload != expectedUpload {
		t.Errorf("Expected %f upload bytes, got %f", expectedUpload, upload)
	}
}

func TestBandwidthTracker_CounterReset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := &BandwidthTracker{logger: logger}

	history1 := &client.HistoryResponse{
		Current:               1000,
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history1)

	// Simulate dishy restart - counter goes backwards
	history2 := &client.HistoryResponse{
		Current:               500, // Reset!
		DownlinkThroughputBps: make([]float64, 900),
		UplinkThroughputBps:   make([]float64, 900),
		PowerIn:               make([]float64, 900),
		PopPingLatencyMs:      make([]float64, 900),
		PopPingDropRate:       make([]float64, 900),
	}
	tracker.processHistory(history2)

	// Counters should not reset, just skip this update
	download, upload := tracker.GetCounters()
	if download != 0 {
		t.Errorf("Expected 0 download after reset, got %f", download)
	}
	if upload != 0 {
		t.Errorf("Expected 0 upload after reset, got %f", upload)
	}
	if tracker.lastCurrent != 500 {
		t.Errorf("Expected lastCurrent=500 after reset, got %d", tracker.lastCurrent)
	}
}
