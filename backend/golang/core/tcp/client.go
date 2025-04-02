package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	defaultBufferSize   = 1024
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// ConnectionStats represents statistics about the connection
type ConnectionStats struct {
	BytesRead    uint64
	BytesWritten uint64
	LastActivity time.Time
	RetryCount   int // Number of retries attempted for read/write operations
}

// Client represents a TCP client with connection management and statistics
type Client struct {
	address      string
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	bufferSize   int
	logger       *log.Logger
	tlsConfig    *tls.Config
	stats        ConnectionStats
	mu           sync.RWMutex
	ctx          context.Context    // Context for the client's lifecycle
	cancel       context.CancelFunc // Cancel function for the client's context
}

// NewClient creates a new TCP client with the given configuration
func NewClient(
	address string,
	tlsConfig *tls.Config,
	opts ...ClientOption,
) (*Client, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}

	// Initialize with a background context. Reconnect might replace this.
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		address:      address,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		bufferSize:   defaultBufferSize,
		tlsConfig:    tlsConfig,
		logger:       log.Default(), // Default logger
		ctx:          ctx,
		cancel:       cancel,
		stats: ConnectionStats{
			LastActivity: time.Now(),
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// Connect establishes a connection to the server
func (c *Client) Connect() error {
	c.mu.RLock()
	// Check if already connected or context cancelled while holding read lock
	if c.conn != nil {
		c.mu.RUnlock()
		return &ConnectionError{Op: "connect", Err: errors.New("already connected")}
	}
	select {
	case <-c.ctx.Done():
		c.mu.RUnlock()
		return &ConnectionError{Op: "connect", Err: fmt.Errorf("client context cancelled: %w", c.ctx.Err())}
	default:
	}
	c.mu.RUnlock()

	// --- Dialing without holding the lock ---
	var conn net.Conn
	var err error
	dialer := net.Dialer{Timeout: c.writeTimeout} // Use writeTimeout as connect timeout, or add a specific connect timeout option

	if c.tlsConfig != nil {
		// Pass context to DialContext for cancellable dialing
		conn, err = tls.DialWithDialer(&dialer, TCP, c.address, c.tlsConfig)
	} else {
		// Pass context to DialContext for cancellable dialing
		conn, err = dialer.DialContext(c.ctx, TCP, c.address)
	}

	if err != nil {
		// Check if context was cancelled during dial
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return wrapError("connect", fmt.Errorf("dial cancelled or timed out: %w", err), false) // Not retryable in this context
		}
		return wrapError("connect", err, true) // Network errors are potentially retryable
	}
	// --- End Dialing ---

	// Lock again to update the connection and stats
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if another goroutine connected successfully or context was cancelled while dialing
	if c.conn != nil {
		conn.Close() // Close the newly created connection as we don't need it
		return &ConnectionError{Op: "connect", Err: errors.New("connection established concurrently")}
	}
	select {
	case <-c.ctx.Done():
		conn.Close() // Close the newly created connection
		return &ConnectionError{Op: "connect", Err: fmt.Errorf("client context cancelled during dial: %w", c.ctx.Err())}
	default:
	}

	c.conn = conn
	c.stats.LastActivity = time.Now()
	// Reset stats for the new connection if needed (e.g., BytesRead/Written)
	// c.stats.BytesRead = 0
	// c.stats.BytesWritten = 0
	c.logger.Printf("Connected to %s", c.address)
	return nil
}

// Read reads data from the connection
func (c *Client) Read() ([]byte, error) {
	c.mu.RLock()
	conn := c.conn // Get current connection under read lock
	c.mu.RUnlock() // Unlock before potentially blocking I/O

	if conn == nil {
		return nil, &ConnectionError{Op: Read, Err: ErrConnectionClosed}
	}

	// Check context cancellation *before* setting deadline and reading
	select {
	case <-c.ctx.Done():
		return nil, &ConnectionError{Op: Read, Err: fmt.Errorf("context cancelled: %w", c.ctx.Err())}
	default:
	}

	if err := conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		// Check if the error is due to using a closed connection, which might happen
		// if Close() was called concurrently after the nil check but before SetReadDeadline.
		if errors.Is(err, net.ErrClosed) {
			return nil, wrapError("set read deadline", ErrConnectionClosed, false)
		}
		return nil, wrapError("set read deadline", err, false)
	}
	// No need to defer reset deadline if connection might be replaced by Reconnect
	// defer conn.SetReadDeadline(time.Time{}) // Reset deadline after read

	buf := make([]byte, c.bufferSize)
	n, err := conn.Read(buf)
	if err != nil {
		// Reset deadline immediately on error to avoid interfering with potential reconnect/close
		conn.SetReadDeadline(time.Time{})

		// Check if the error is due to context cancellation (e.g., timeout triggered deadline)
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			select {
			case <-c.ctx.Done():
				return nil, &ConnectionError{Op: Read, Err: fmt.Errorf("context cancelled: %w", c.ctx.Err())}
			default:
				// It was a genuine read timeout
				return nil, wrapError(Read, ErrTimeout, true) // Timeout is often retryable
			}
		}
		// Check if the connection was closed
		if errors.Is(err, net.ErrClosed) {
			return nil, wrapError(Read, ErrConnectionClosed, false)
		}
		return nil, wrapError(Read, err, isNetworkErrorRetryable(err)) // Wrap other errors
	}

	// Reset deadline after successful read
	conn.SetReadDeadline(time.Time{})

	c.mu.Lock()
	// Ensure we are updating stats for the *same* connection we read from.
	// This check is mostly relevant if Reconnect can happen very rapidly.
	if c.conn == conn {
		c.stats.BytesRead += uint64(n)
		c.stats.LastActivity = time.Now()
	}
	c.mu.Unlock()
	return buf[:n], nil
}

// Write writes data to the connection
func (c *Client) Write(data []byte) error {
	c.mu.RLock()
	conn := c.conn // Get current connection under read lock
	c.mu.RUnlock() // Unlock before potentially blocking I/O

	if conn == nil {
		return &ConnectionError{Op: Write, Err: ErrConnectionClosed}
	}

	// Check context cancellation *before* setting deadline and writing
	select {
	case <-c.ctx.Done():
		return &ConnectionError{Op: Write, Err: fmt.Errorf("context cancelled: %w", c.ctx.Err())}
	default:
	}

	if err := conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		if errors.Is(err, net.ErrClosed) {
			return wrapError("set write deadline", ErrConnectionClosed, false)
		}
		return wrapError("set write deadline", err, false)
	}
	// No need to defer reset deadline

	n, err := conn.Write(data)
	if err != nil {
		// Reset deadline immediately on error
		conn.SetWriteDeadline(time.Time{})

		// Check for timeout / context cancellation
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			select {
			case <-c.ctx.Done():
				return &ConnectionError{Op: Write, Err: fmt.Errorf("context cancelled: %w", c.ctx.Err())}
			default:
				return wrapError(Write, ErrTimeout, true) // Timeout is retryable
			}
		}
		if errors.Is(err, net.ErrClosed) {
			return wrapError(Write, ErrConnectionClosed, false)
		}
		return wrapError(Write, err, isNetworkErrorRetryable(err)) // Wrap other errors
	}

	// Reset deadline after successful write
	conn.SetWriteDeadline(time.Time{})

	c.mu.Lock()
	// Update stats only if the connection hasn't changed
	if c.conn == conn {
		c.stats.BytesWritten += uint64(n)
		c.stats.LastActivity = time.Now()
	}
	c.mu.Unlock()
	return nil
}

// Close closes the connection and cancels the client's context.
func (c *Client) Close() error {
	c.mu.Lock() // Acquire write lock
	// Cancel context first to signal ongoing operations (Read/Write/Connect/Retry)
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil // Prevent double cancel
	}

	conn := c.conn
	c.conn = nil  // Set internal connection to nil immediately
	c.mu.Unlock() // Release lock before closing connection

	if conn == nil {
		c.logger.Printf("Close called but connection was already nil")
		return nil // Nil seems more idempotent.
	}

	err := conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		c.logger.Printf("Error closing connection: %v", err)
		return wrapError("close", err, false)
	}

	c.logger.Printf("Connection closed")
	return nil
}

// Reconnect closes the current connection, creates a new context, and establishes a new connection.
func (c *Client) Reconnect() error {
	c.logger.Printf("Reconnect requested")
	// Close the existing connection and cancel its associated context first.
	// Ignore the error from Close() for now, as we are trying to establish a new connection anyway.
	_ = c.Close()

	// --- Create a new context for the new connection lifecycle ---
	c.mu.Lock()
	// Check if already cancelled externally before creating new context
	// This check might be overly cautious depending on use case
	/*
		if c.ctx.Err() != nil {
			c.mu.Unlock()
			return fmt.Errorf("cannot reconnect, client context is already cancelled: %w", c.ctx.Err())
		}
	*/
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()
	// --- End new context creation ---

	// Attempt to connect using the new context.
	// Connect() handles locking internally.
	err := c.Connect()
	if err != nil {
		// If connect fails, cancel the context we just created
		c.mu.Lock()
		if c.cancel != nil {
			c.cancel()
			c.cancel = nil
		}
		c.mu.Unlock()
		return fmt.Errorf("reconnect failed during connect: %w", err)
	}
	return nil
}

// RemoteAddr returns the remote network address.
func (c *Client) RemoteAddr() net.Addr {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

// Stats returns the current connection statistics.
func (c *Client) Stats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to avoid race conditions if the caller modifies it
	// (though ConnectionStats fields are basic types, so direct return is usually safe)
	statsCopy := c.stats
	return statsCopy
}
