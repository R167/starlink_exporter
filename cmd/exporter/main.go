package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/R167/starlink_exporter/internal/client"
	"github.com/R167/starlink_exporter/internal/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddr = flag.String("listen", ":9999", "Address to listen on for metrics")
	dishAddr   = flag.String("dish", "192.168.100.1:9200", "Starlink dish gRPC address")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// Setup structured logging
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create native gRPC client
	grpcClient, err := client.NewNativeGRPCClient(*dishAddr)
	if err != nil {
		logger.Error("Failed to create gRPC client", "error", err)
		os.Exit(1)
	}
	defer grpcClient.Close()

	// Create and start bandwidth tracker
	bandwidthTracker := collector.NewBandwidthTracker(grpcClient, logger)
	go bandwidthTracker.Start(ctx)

	// Create and register Starlink collector
	starlinkCollector := collector.NewStarlinkCollector(grpcClient, bandwidthTracker, logger)
	prometheus.MustRegister(starlinkCollector)

	// Setup HTTP server with timeouts
	http.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		logger.Info("Starting Starlink exporter", "address", *listenAddr, "dish", *dishAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			cancel() // Cancel context before exit
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, stopping gracefully...")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// Stop bandwidth tracker
	cancel()
	bandwidthTracker.Stop()

	logger.Info("Exporter stopped")
}
