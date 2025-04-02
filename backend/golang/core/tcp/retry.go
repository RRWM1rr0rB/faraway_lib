package tcp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// WriteWithRetry attempts to write data, retrying on failure with reconnection.
func (c *Client) WriteWithRetry(data []byte, maxRetries int, backoff time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// Check context cancellation before attempting write
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("write cancelled by context: %w", c.ctx.Err())
		default:
		}

		err := c.Write(data)
		if err == nil {
			c.mu.Lock()
			c.stats.RetryCount = 0 // Reset retry count on success
			c.mu.Unlock()
			return nil // Success
		}

		lastErr = err
		c.logger.Printf("Write attempt %d/%d failed: %v. Retrying in %v...", i+1, maxRetries, err, backoff)
		c.mu.Lock()
		c.stats.RetryCount++ // Increment retry count on failure
		c.mu.Unlock()

		// Check context cancellation before sleeping
		select {
		case <-time.After(backoff):
			// Continue after backoff
		case <-c.ctx.Done():
			return fmt.Errorf("retry cancelled by context: %w", c.ctx.Err())
		}

		// Check if error is potentially recoverable by reconnecting
		var connErr *ConnectionError
		if errors.As(err, &connErr) && connErr.IsRetryable || errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			c.logger.Printf("Attempting to reconnect...")
			reconnectErr := c.Reconnect() // Reconnect now uses the new context
			if reconnectErr != nil {
				c.logger.Printf("Reconnect failed: %v", reconnectErr)
				// If reconnect fails, maybe return the reconnect error or the original write error
				// Returning the original error might be more informative about the root cause
				return fmt.Errorf("reconnect after write failed: %w (original write error: %v)", reconnectErr, lastErr)
			}
			c.logger.Printf("Reconnect successful.")
			// Optional: Implement exponential backoff here instead of fixed backoff
		} else {
			// If the error is not retryable (e.g., bad data), don't retry/reconnect
			c.logger.Printf("Non-retryable write error: %v", err)
			return lastErr
		}
	}

	return fmt.Errorf("write failed after %d retries: %w", maxRetries, lastErr)
}

// ReadWithRetry attempts to read data, retrying on failure with reconnection.
func (c *Client) ReadWithRetry(maxRetries int, backoff time.Duration) ([]byte, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// Check context cancellation before attempting read
		select {
		case <-c.ctx.Done():
			return nil, fmt.Errorf("read cancelled by context: %w", c.ctx.Err())
		default:
		}

		data, err := c.Read()
		if err == nil {
			c.mu.Lock()
			c.stats.RetryCount = 0 // Reset retry count on success
			c.mu.Unlock()
			return data, nil // Success
		}

		lastErr = err
		c.logger.Printf("Read attempt %d/%d failed: %v. Retrying in %v...", i+1, maxRetries, err, backoff)
		c.mu.Lock()
		c.stats.RetryCount++ // Increment retry count on failure
		c.mu.Unlock()

		// Check if the error suggests a broken connection that might be fixed by reconnecting
		isBrokenPipe := false
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// Common errors indicating broken connection
			if opErr.Err.Error() == "read: connection reset by peer" || opErr.Err.Error() == "use of closed network connection" {
				isBrokenPipe = true
			}
		}
		// Also check for io.EOF, ErrConnectionClosed, and retryable network errors
		if errors.Is(err, io.EOF) || errors.Is(err, ErrConnectionClosed) || isBrokenPipe || (errors.As(err, new(*ConnectionError)) && (*new(*ConnectionError)).IsRetryable) {
			// Check context cancellation before sleeping
			select {
			case <-time.After(backoff):
				// Continue after backoff
			case <-c.ctx.Done():
				return nil, fmt.Errorf("retry cancelled by context: %w", c.ctx.Err())
			}

			c.logger.Printf("Attempting to reconnect...")
			reconnectErr := c.Reconnect() // Reconnect now uses the new context
			if reconnectErr != nil {
				c.logger.Printf("Reconnect failed: %v", reconnectErr)
				return nil, fmt.Errorf("reconnect after read failed: %w (original read error: %v)", reconnectErr, lastErr)
			}
			c.logger.Printf("Reconnect successful.")
			// Optional: Implement exponential backoff here
		} else {
			// If error is not related to connection state, just wait and retry read
			c.logger.Printf("Read error potentially not connection related, retrying read without reconnect: %v", err)
			// Check context cancellation before sleeping
			select {
			case <-time.After(backoff):
				// Continue after backoff
			case <-c.ctx.Done():
				return nil, fmt.Errorf("retry cancelled by context: %w", c.ctx.Err())
			}
		}
	}

	return nil, fmt.Errorf("read failed after %d retries: %w", maxRetries, lastErr)
}
