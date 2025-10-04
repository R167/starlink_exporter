// Package collector implements Prometheus collectors for Starlink metrics.
//
// The collector package provides a background bandwidth tracker that continuously
// monitors Starlink dish metrics via gRPC and integrates them into Prometheus counters.
// It handles circular buffer processing from the dish's history API to accurately
// track bandwidth, energy consumption, and ping statistics.
package collector
