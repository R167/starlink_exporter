package client

// Client interface for Starlink dish communication
type Client interface {
	GetStatus() (*StatusResponse, error)
	GetHistory() (*HistoryResponse, error)
}

// DeviceInfo contains device information
type DeviceInfo struct {
	ID              string `json:"id"`
	HardwareVersion string `json:"hardwareVersion"`
	SoftwareVersion string `json:"softwareVersion"`
	CountryCode     string `json:"countryCode"`
	BootCount       int    `json:"bootcount"`
}

// DeviceState contains device state
type DeviceState struct {
	UptimeS uint64 `json:"uptimeS"`
}

// ObstructionStats contains obstruction statistics
type ObstructionStats struct {
	FractionObstructed float64 `json:"fractionObstructed"`
	ValidS             float64 `json:"validS"`
	TimeObstructed     float64 `json:"timeObstructed"`
}

// GPSStats contains GPS statistics
type GPSStats struct {
	GPSValid bool `json:"gpsValid"`
	GPSSats  int  `json:"gpsSats"`
}

// StatusResponse contains status data from the dish
type StatusResponse struct {
	DeviceInfo            DeviceInfo       `json:"deviceInfo"`
	DeviceState           DeviceState      `json:"deviceState"`
	ObstructionStats      ObstructionStats `json:"obstructionStats"`
	DownlinkThroughputBps float64          `json:"downlinkThroughputBps"`
	UplinkThroughputBps   float64          `json:"uplinkThroughputBps"`
	PopPingLatencyMs      float64          `json:"popPingLatencyMs"`
	BoresightAzimuthDeg   float64          `json:"boresightAzimuthDeg"`
	BoresightElevationDeg float64          `json:"boresightElevationDeg"`
	GPSStats              GPSStats         `json:"gpsStats"`
	EthSpeedMbps          int              `json:"ethSpeedMbps"`
	IsSnrAboveNoiseFloor  bool             `json:"isSnrAboveNoiseFloor"`
}

// HistoryResponse contains historical data from the dish
type HistoryResponse struct {
	Current               uint64    `json:"current"`
	DownlinkThroughputBps []float64 `json:"downlinkThroughputBps"`
	UplinkThroughputBps   []float64 `json:"uplinkThroughputBps"`
	PopPingLatencyMs      []float64 `json:"popPingLatencyMs"`
	PopPingDropRate       []float64 `json:"popPingDropRate"`
	PowerIn               []float64 `json:"powerIn"`
}
