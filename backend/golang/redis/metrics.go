package redis

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/RRWM1rr0rB/faraway_lib/backend/golang/metrics"
)

const (
	defaultCheckInterval = 10 * time.Second
	maxConsecutiveErrors = 5
)

var (
	redisAvailability = metrics.NewGaugeVec(
		metrics.GaugeOpts{
			Name: "redis_availability",
			Help: "Redis connection availability (1 - available, 0 - unavailable)",
		},
		[]string{"address", "db"},
	)

	redisLatency = metrics.NewHistogramVec(
		metrics.HistogramOpts{
			Name:    "redis_ping_latency_seconds",
			Help:    "Redis ping latency distribution",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		},
		[]string{"address", "db"},
	)
)

// startHealthCheck initiates periodic Redis health checks
func startHealthCheck(ctx context.Context, client *Client, cfg *Config) {
	labels := []string{cfg.address, strconv.Itoa(cfg.db)}
	consecutiveErrors := 0

	// Initialize metrics
	redisAvailability.WithLabelValues(labels...).Set(0)

	// Initial delay for first check
	time.Sleep(time.Second * 2)

	go func() {
		defer handlePanic("health check goroutine")

		ticker := time.NewTicker(cfg.GetHealthCheckInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				status, latency := checkConnection(ctx, client)
				updateMetrics(labels, status, latency)
				updateHealthStatus(cfg, status)

				if !status {
					consecutiveErrors++
					logError(cfg.address, consecutiveErrors)
				} else {
					consecutiveErrors = 0
				}

			case <-ctx.Done():
				log.Printf("Stopping health checks for Redis: %s", cfg.address)
				return
			}
		}
	}()
}

// checkConnection performs Redis ping with timeout
func checkConnection(ctx context.Context, client *Client) (bool, time.Duration) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	return err == nil, time.Since(start)
}

// updateMetrics updates Prometheus metrics
func updateMetrics(labels []string, available bool, latency time.Duration) {
	status := 0.0
	if available {
		status = 1.0
	}

	redisAvailability.WithLabelValues(labels...).Set(status)
	redisLatency.WithLabelValues(labels...).Observe(latency.Seconds())
}

// updateHealthStatus updates external health checker
func updateHealthStatus(cfg *Config, available bool) {
	if cfg.health.checker != nil {
		name := cfg.health.name
		if name == "" {
			name = defaultName
		}
		cfg.health.checker.SetStatus(name, available)
	}
}

// logError implements error logging with throttling
func logError(address string, count int) {
	switch {
	case count >= maxConsecutiveErrors:
		log.Printf("[REDIS CRITICAL] Persistent connection issues with %s (%d failures)", address, count)
	case count%maxConsecutiveErrors == 0:
		log.Printf("[REDIS ERROR] Connection issues with %s (%d attempts)", address, count)
	}
}

// handlePanic recovers from goroutine panics
func handlePanic(source string) {
	if r := recover(); r != nil {
		log.Printf("[PANIC] Recovered in %s: %v", source, r)
	}
}
