package collector

import (
	"log/slog"

	"github.com/R167/starlink_exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

// StarlinkCollector collects metrics from Starlink dish
type StarlinkCollector struct {
	client           client.Client
	logger           *slog.Logger
	bandwidthTracker *BandwidthTracker

	// Counters
	downloadBytesTotal      *prometheus.Desc
	uploadBytesTotal        *prometheus.Desc
	energyJoulesTotal       *prometheus.Desc
	pingLatencySecondsSum   *prometheus.Desc
	pingLatencySecondsCount *prometheus.Desc
	pingDropTotal           *prometheus.Desc

	// Gauges - Current Status
	downlinkThroughputBps *prometheus.Desc
	uplinkThroughputBps   *prometheus.Desc
	popPingLatencyMs      *prometheus.Desc
	uptimeSeconds         *prometheus.Desc
	obstructionFraction   *prometheus.Desc
	obstructionValidS     *prometheus.Desc
	gpsSats               *prometheus.Desc
	gpsValid              *prometheus.Desc
	ethSpeedMbps          *prometheus.Desc
	snrAboveNoiseFloor    *prometheus.Desc

	// Status
	up *prometheus.Desc

	// Info metric
	info *prometheus.Desc
}

// NewStarlinkCollector creates a new Starlink collector
func NewStarlinkCollector(c client.Client, tracker *BandwidthTracker, logger *slog.Logger) *StarlinkCollector {
	return &StarlinkCollector{
		client:           c,
		logger:           logger,
		bandwidthTracker: tracker,

		// Counters
		downloadBytesTotal: prometheus.NewDesc(
			"starlink_download_bytes_total",
			"Total bytes downloaded",
			nil, nil,
		),
		uploadBytesTotal: prometheus.NewDesc(
			"starlink_upload_bytes_total",
			"Total bytes uploaded",
			nil, nil,
		),
		energyJoulesTotal: prometheus.NewDesc(
			"starlink_energy_joules_total",
			"Total energy consumed (joules)",
			nil, nil,
		),
		pingLatencySecondsSum: prometheus.NewDesc(
			"starlink_ping_latency_seconds_sum",
			"Sum of ping latencies in seconds (summary metric)",
			nil, nil,
		),
		pingLatencySecondsCount: prometheus.NewDesc(
			"starlink_ping_latency_seconds_count",
			"Count of ping samples (summary metric)",
			nil, nil,
		),
		pingDropTotal: prometheus.NewDesc(
			"starlink_ping_drop_total",
			"Total ping drops",
			nil, nil,
		),

		// Gauges
		downlinkThroughputBps: prometheus.NewDesc(
			"starlink_downlink_throughput_bps",
			"Current downlink throughput in bits per second",
			nil, nil,
		),
		uplinkThroughputBps: prometheus.NewDesc(
			"starlink_uplink_throughput_bps",
			"Current uplink throughput in bits per second",
			nil, nil,
		),
		popPingLatencyMs: prometheus.NewDesc(
			"starlink_pop_ping_latency_ms",
			"Current ping latency to POP in milliseconds",
			nil, nil,
		),
		uptimeSeconds: prometheus.NewDesc(
			"starlink_uptime_seconds",
			"Device uptime in seconds",
			nil, nil,
		),
		obstructionFraction: prometheus.NewDesc(
			"starlink_obstruction_fraction",
			"Fraction of time obstructed",
			nil, nil,
		),
		obstructionValidS: prometheus.NewDesc(
			"starlink_obstruction_valid_seconds",
			"Valid observation time for obstruction stats",
			nil, nil,
		),
		gpsSats: prometheus.NewDesc(
			"starlink_gps_satellites",
			"Number of GPS satellites",
			nil, nil,
		),
		gpsValid: prometheus.NewDesc(
			"starlink_gps_valid",
			"GPS validity (1 = valid, 0 = invalid)",
			nil, nil,
		),
		ethSpeedMbps: prometheus.NewDesc(
			"starlink_eth_speed_mbps",
			"Ethernet speed in Mbps",
			nil, nil,
		),
		snrAboveNoiseFloor: prometheus.NewDesc(
			"starlink_snr_above_noise_floor",
			"SNR above noise floor (1 = yes, 0 = no)",
			nil, nil,
		),

		// Status
		up: prometheus.NewDesc(
			"starlink_up",
			"Whether the last scrape of Starlink metrics was successful (1 = success, 0 = failure)",
			nil, nil,
		),

		// Info
		info: prometheus.NewDesc(
			"starlink_info",
			"Starlink device information",
			[]string{"id", "hardware_version", "software_version", "country_code"}, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *StarlinkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.downloadBytesTotal
	ch <- c.uploadBytesTotal
	ch <- c.energyJoulesTotal
	ch <- c.pingLatencySecondsSum
	ch <- c.pingLatencySecondsCount
	ch <- c.pingDropTotal
	ch <- c.downlinkThroughputBps
	ch <- c.uplinkThroughputBps
	ch <- c.popPingLatencyMs
	ch <- c.uptimeSeconds
	ch <- c.obstructionFraction
	ch <- c.obstructionValidS
	ch <- c.gpsSats
	ch <- c.gpsValid
	ch <- c.ethSpeedMbps
	ch <- c.snrAboveNoiseFloor
	ch <- c.up
	ch <- c.info
}

// Collect implements prometheus.Collector
func (c *StarlinkCollector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debug("Prometheus scrape started")

	// Get status
	status, err := c.client.GetStatus()
	if err != nil {
		c.logger.Error("Failed to get status", "error", err)
		// Emit up=0 to indicate failure
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0.0)
		// Still emit all counters even on error
		download, upload := c.bandwidthTracker.GetCounters()
		ch <- prometheus.MustNewConstMetric(c.downloadBytesTotal, prometheus.CounterValue, download)
		ch <- prometheus.MustNewConstMetric(c.uploadBytesTotal, prometheus.CounterValue, upload)

		energy := c.bandwidthTracker.GetEnergyJoules()
		ch <- prometheus.MustNewConstMetric(c.energyJoulesTotal, prometheus.CounterValue, energy)

		pingLatencySum, pingSampleCount, pingDrops := c.bandwidthTracker.GetPingMetrics()
		ch <- prometheus.MustNewConstMetric(c.pingLatencySecondsSum, prometheus.CounterValue, pingLatencySum)
		ch <- prometheus.MustNewConstMetric(c.pingLatencySecondsCount, prometheus.CounterValue, pingSampleCount)
		ch <- prometheus.MustNewConstMetric(c.pingDropTotal, prometheus.CounterValue, pingDrops)

		c.logger.Debug("Prometheus scrape completed (error path)", "download_bytes", download, "upload_bytes", upload)
		return
	}

	// Emit up=1 for successful scrape
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1.0)

	// Counters from background tracker
	download, upload := c.bandwidthTracker.GetCounters()
	ch <- prometheus.MustNewConstMetric(
		c.downloadBytesTotal,
		prometheus.CounterValue,
		download,
	)
	ch <- prometheus.MustNewConstMetric(
		c.uploadBytesTotal,
		prometheus.CounterValue,
		upload,
	)

	energy := c.bandwidthTracker.GetEnergyJoules()
	ch <- prometheus.MustNewConstMetric(
		c.energyJoulesTotal,
		prometheus.CounterValue,
		energy,
	)

	pingLatencySum, pingSampleCount, pingDrops := c.bandwidthTracker.GetPingMetrics()
	ch <- prometheus.MustNewConstMetric(
		c.pingLatencySecondsSum,
		prometheus.CounterValue,
		pingLatencySum,
	)
	ch <- prometheus.MustNewConstMetric(
		c.pingLatencySecondsCount,
		prometheus.CounterValue,
		pingSampleCount,
	)
	ch <- prometheus.MustNewConstMetric(
		c.pingDropTotal,
		prometheus.CounterValue,
		pingDrops,
	)

	// Gauges - Current throughput
	ch <- prometheus.MustNewConstMetric(
		c.downlinkThroughputBps,
		prometheus.GaugeValue,
		status.DownlinkThroughputBps,
	)
	ch <- prometheus.MustNewConstMetric(
		c.uplinkThroughputBps,
		prometheus.GaugeValue,
		status.UplinkThroughputBps,
	)

	// Latency
	ch <- prometheus.MustNewConstMetric(
		c.popPingLatencyMs,
		prometheus.GaugeValue,
		status.PopPingLatencyMs,
	)

	// Uptime
	ch <- prometheus.MustNewConstMetric(
		c.uptimeSeconds,
		prometheus.GaugeValue,
		float64(status.DeviceState.UptimeS),
	)

	// Obstruction stats
	ch <- prometheus.MustNewConstMetric(
		c.obstructionFraction,
		prometheus.GaugeValue,
		status.ObstructionStats.FractionObstructed,
	)
	ch <- prometheus.MustNewConstMetric(
		c.obstructionValidS,
		prometheus.GaugeValue,
		status.ObstructionStats.ValidS,
	)

	// GPS stats
	ch <- prometheus.MustNewConstMetric(
		c.gpsSats,
		prometheus.GaugeValue,
		float64(status.GPSStats.GPSSats),
	)
	gpsValidValue := 0.0
	if status.GPSStats.GPSValid {
		gpsValidValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		c.gpsValid,
		prometheus.GaugeValue,
		gpsValidValue,
	)

	// Ethernet speed
	ch <- prometheus.MustNewConstMetric(
		c.ethSpeedMbps,
		prometheus.GaugeValue,
		float64(status.EthSpeedMbps),
	)

	// SNR
	snrValue := 0.0
	if status.IsSnrAboveNoiseFloor {
		snrValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		c.snrAboveNoiseFloor,
		prometheus.GaugeValue,
		snrValue,
	)

	// Info metric with labels
	ch <- prometheus.MustNewConstMetric(
		c.info,
		prometheus.GaugeValue,
		1.0,
		status.DeviceInfo.ID,
		status.DeviceInfo.HardwareVersion,
		status.DeviceInfo.SoftwareVersion,
		status.DeviceInfo.CountryCode,
	)

	c.logger.Debug("Prometheus scrape completed", "download_bytes", download, "upload_bytes", upload)
}
