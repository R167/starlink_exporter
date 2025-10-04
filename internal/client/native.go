package client

import (
	"context"
	"fmt"
	"time"

	pb "github.com/R167/starlink_exporter/proto/spacex_api/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NativeGRPCClient uses generated protobuf code for gRPC communication
type NativeGRPCClient struct {
	conn   *grpc.ClientConn
	client pb.DeviceClient
}

// NewNativeGRPCClient creates a new native gRPC client
func NewNativeGRPCClient(address string) (*NativeGRPCClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return &NativeGRPCClient{
		conn:   conn,
		client: pb.NewDeviceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *NativeGRPCClient) Close() error {
	return c.conn.Close()
}

// GetStatus retrieves current status from the dish
func (c *NativeGRPCClient) GetStatus() (*StatusResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.Request{
		Request: &pb.Request_GetStatus{
			GetStatus: &pb.GetStatusRequest{},
		},
	}

	resp, err := c.client.Handle(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc failed: %v", err)
	}

	dishStatus := resp.GetDishGetStatus()
	if dishStatus == nil {
		return nil, fmt.Errorf("no dish status in response")
	}

	return &StatusResponse{
		DeviceInfo: DeviceInfo{
			ID:              dishStatus.DeviceInfo.Id,
			HardwareVersion: dishStatus.DeviceInfo.HardwareVersion,
			SoftwareVersion: dishStatus.DeviceInfo.SoftwareVersion,
			CountryCode:     dishStatus.DeviceInfo.CountryCode,
			BootCount:       int(dishStatus.DeviceInfo.Bootcount),
		},
		DeviceState: DeviceState{
			UptimeS: dishStatus.DeviceState.UptimeS,
		},
		ObstructionStats: ObstructionStats{
			FractionObstructed: float64(dishStatus.ObstructionStats.FractionObstructed),
			ValidS:             float64(dishStatus.ObstructionStats.ValidS),
			TimeObstructed:     float64(dishStatus.ObstructionStats.TimeObstructed),
		},
		DownlinkThroughputBps: float64(dishStatus.DownlinkThroughputBps),
		UplinkThroughputBps:   float64(dishStatus.UplinkThroughputBps),
		PopPingLatencyMs:      float64(dishStatus.PopPingLatencyMs),
		BoresightAzimuthDeg:   float64(dishStatus.BoresightAzimuthDeg),
		BoresightElevationDeg: float64(dishStatus.BoresightElevationDeg),
		GPSStats: GPSStats{
			GPSValid: dishStatus.GpsStats.GpsValid,
			GPSSats:  int(dishStatus.GpsStats.GpsSats),
		},
		EthSpeedMbps:         int(dishStatus.EthSpeedMbps),
		IsSnrAboveNoiseFloor: dishStatus.IsSnrAboveNoiseFloor,
	}, nil
}

// GetHistory retrieves historical data from the dish
func (c *NativeGRPCClient) GetHistory() (*HistoryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.Request{
		Request: &pb.Request_GetHistory{
			GetHistory: &pb.GetHistoryRequest{},
		},
	}

	resp, err := c.client.Handle(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc failed: %v", err)
	}

	dishHistory := resp.GetDishGetHistory()
	if dishHistory == nil {
		return nil, fmt.Errorf("no dish history in response")
	}

	// Convert to float64 arrays
	downlink := make([]float64, len(dishHistory.DownlinkThroughputBps))
	for i, v := range dishHistory.DownlinkThroughputBps {
		downlink[i] = float64(v)
	}

	uplink := make([]float64, len(dishHistory.UplinkThroughputBps))
	for i, v := range dishHistory.UplinkThroughputBps {
		uplink[i] = float64(v)
	}

	popPingLatency := make([]float64, len(dishHistory.PopPingLatencyMs))
	for i, v := range dishHistory.PopPingLatencyMs {
		popPingLatency[i] = float64(v)
	}

	popPingDropRate := make([]float64, len(dishHistory.PopPingDropRate))
	for i, v := range dishHistory.PopPingDropRate {
		popPingDropRate[i] = float64(v)
	}

	powerIn := make([]float64, len(dishHistory.PowerIn))
	for i, v := range dishHistory.PowerIn {
		powerIn[i] = float64(v)
	}

	return &HistoryResponse{
		Current:               dishHistory.Current,
		DownlinkThroughputBps: downlink,
		UplinkThroughputBps:   uplink,
		PopPingLatencyMs:      popPingLatency,
		PopPingDropRate:       popPingDropRate,
		PowerIn:               powerIn,
	}, nil
}
