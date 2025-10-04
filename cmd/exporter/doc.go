// Package main implements the Starlink Prometheus exporter.
//
// This exporter connects to a Starlink dish via gRPC and exports metrics
// to Prometheus on port 9999 (configurable). It tracks bandwidth, energy
// consumption, ping latency, and device status by processing circular buffer
// history data from the dish every second.
package main
