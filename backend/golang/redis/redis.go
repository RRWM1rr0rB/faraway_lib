package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
)

const (
	defaultIntervalCheck       = 10 * time.Second
	defaultName                = "redis"
	defaultMaxRetries          = 5
	defaultPoolSize            = 20
	defaultMinIdleConns        = 5
	defaultDialTimeout         = 5 * time.Second
	defaultReadTimeout         = 3 * time.Second
	defaultWriteTimeout        = 3 * time.Second
	defaultHealthCheckInt      = 15 * time.Second
	defaultHealthCheckInterval = 15 * time.Second
)

// HealthChecker defines health status reporting interface
type HealthChecker interface {
	SetStatus(name string, status bool)
}

// Config holds Redis connection parameters
type Config struct {
	address             string
	password            string
	db                  int
	isTLS               bool
	dialTimeout         time.Duration
	readTimeout         time.Duration
	writeTimeout        time.Duration
	poolSize            int
	minIdleConns        int
	maxRetries          int
	healthCheckInt      time.Duration
	healthCheckInterval time.Duration
	breakerConfig       gobreaker.Settings
	health              struct {
		checker HealthChecker
		name    string
	}
}

// Options configures Redis client
type Options func(*Config)

// Client wraps redis.Client with production-grade features
type Client struct {
	*redis.Client
	breaker   *gobreaker.CircuitBreaker
	cmdLogger redis.Hook
}

// NewRedisConfig creates a validated Config instance
func NewRedisConfig(
	address string,
	password string,
	db int,
	isTLS bool,
	opts ...Options,
) *Config {
	cfg := &Config{
		address:        address,
		password:       password,
		db:             db,
		isTLS:          isTLS,
		dialTimeout:    defaultDialTimeout,
		readTimeout:    defaultReadTimeout,
		writeTimeout:   defaultWriteTimeout,
		poolSize:       defaultPoolSize,
		minIdleConns:   defaultMinIdleConns,
		maxRetries:     defaultMaxRetries,
		healthCheckInt: defaultHealthCheckInt,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithTimeoutConfig sets connection timeouts
func WithTimeoutConfig(dial, read, write time.Duration) Options {
	return func(cfg *Config) {
		cfg.dialTimeout = dial
		cfg.readTimeout = read
		cfg.writeTimeout = write
	}
}

func (c *Config) GetHealthCheckInterval() time.Duration {
	if c.healthCheckInterval <= 0 {
		return defaultHealthCheckInterval
	}
	return c.healthCheckInterval
}

// WithPoolConfig configures connection pool
func WithPoolConfig(size, minIdle int) Options {
	return func(cfg *Config) {
		cfg.poolSize = size
		cfg.minIdleConns = minIdle
	}
}

// WithCircuitBreaker configures circuit breaker
func WithCircuitBreaker(settings gobreaker.Settings) Options {
	return func(cfg *Config) {
		cfg.breakerConfig = settings
	}
}

// WithHealthCheckInterval configures health check frequency
func WithHealthCheckInterval(interval time.Duration) Options {
	return func(cfg *Config) {
		cfg.healthCheckInt = interval
	}
}

// WithHealthChecker sets health monitoring
func WithHealthChecker(name string, hc HealthChecker) Options {
	return func(cfg *Config) {
		cfg.health.checker = hc
		cfg.health.name = name
	}
}

// NewClient creates a robust Redis client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	options := &redis.Options{
		Addr:         cfg.address,
		Password:     cfg.password,
		DB:           cfg.db,
		DialTimeout:  cfg.dialTimeout,
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
		PoolSize:     cfg.poolSize,
		MinIdleConns: cfg.minIdleConns,
	}

	if cfg.isTLS {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	baseClient := redis.NewClient(options)
	enhancedClient := &Client{
		Client: baseClient,
	}

	// Add command logging
	enhancedClient.cmdLogger = createCommandLoggerHook()
	baseClient.AddHook(enhancedClient.cmdLogger)

	// Configure circuit breaker
	if cfg.breakerConfig.Name == "" {
		cfg.breakerConfig.Name = defaultName
	}
	enhancedClient.breaker = gobreaker.NewCircuitBreaker(cfg.breakerConfig)

	// Connection with retry logic
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = time.Duration(cfg.maxRetries) * time.Second

	err := backoff.Retry(func() error {
		_, err := enhancedClient.breaker.Execute(func() (interface{}, error) {
			return nil, enhancedClient.Ping(ctx).Err()
		})
		return err
	}, backoff.WithContext(backoffConfig, ctx))

	if err != nil {
		return nil, fmt.Errorf("connection failed after %d attempts: %w", cfg.maxRetries, err)
	}

	// Start health checks
	go startHealthCheck(ctx, enhancedClient, cfg)

	return enhancedClient, nil
}

// Close gracefully terminates the client
func (c *Client) Close() error {
	if c.breaker != nil {
		c.breaker = nil
	}
	return c.Client.Close()
}

// createCommandLoggerHook enables detailed command logging
func createCommandLoggerHook() redis.Hook {
	return &loggingHook{}
}

// loggingHook implements redis.Hook interface
type loggingHook struct{}

func (h *loggingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		log.Printf("Connecting to %s://%s", network, addr)
		return next(ctx, network, addr), nil
	}
}

func (h *loggingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		log.Printf("CMD: %v", cmd)
		err := next(ctx, cmd)
		log.Printf("CMD %v completed in %v (error: %v)", cmd.Name(), time.Since(start), err)
		return err
	}
}

func (h *loggingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		log.Printf("PIPELINE: %d commands", len(cmds))
		err := next(ctx, cmds)
		log.Printf("PIPELINE completed in %v (error: %v)", time.Since(start), err)
		return err
	}
}
