package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

const (
	dishAddress = "192.168.100.1:9200"
	service     = "SpaceX.API.Device.Device/Handle"
)

// Client interface for Starlink dish communication
type Client interface {
	GetStatus() (*StatusResponse, error)
	GetHistory() (*HistoryResponse, error)
}

// StarlinkClient interacts with Starlink dish via grpcurl
type StarlinkClient struct {
	address string
}

// NewStarlinkClient creates a new Starlink client
func NewStarlinkClient() *StarlinkClient {
	return &StarlinkClient{
		address: dishAddress,
	}
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

// GetStatus retrieves current status from the dish
func (c *StarlinkClient) GetStatus() (*StatusResponse, error) {
	cmd := exec.Command("grpcurl", "-plaintext", "-d", `{"get_status":{}}`, c.address, service)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("grpcurl failed: %v, stderr: %s", err, stderr.String())
	}

	var response struct {
		DishGetStatus StatusResponse `json:"dishGetStatus"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %v", err)
	}

	return &response.DishGetStatus, nil
}

// GetHistory retrieves historical data from the dish
func (c *StarlinkClient) GetHistory() (*HistoryResponse, error) {
	cmd := exec.Command("grpcurl", "-plaintext", "-d", `{"get_history":{}}`, c.address, service)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("grpcurl failed: %v, stderr: %s", err, stderr.String())
	}

	var response struct {
		DishGetHistory HistoryResponse `json:"dishGetHistory"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse history response: %v", err)
	}

	return &response.DishGetHistory, nil
}
