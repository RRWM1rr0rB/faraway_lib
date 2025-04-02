package tcp

import (
	"errors"
	"io"
	"log"
	"net"
	"time"
)

type ConnectionPool struct {
	factory func() (*Client, error)
	pool    chan *Client
	maxSize int
	logger  *log.Logger
	// Add a timeout for the ping check
	pingTimeout time.Duration
}

func NewConnectionPool(factory func() (*Client, error), maxSize int) *ConnectionPool {
	// Consider adding options, e.g., for pingTimeout
	pingTimeout := 1 * time.Second // Default ping timeout
	return &ConnectionPool{
		factory:     factory,
		pool:        make(chan *Client, maxSize),
		maxSize:     maxSize,
		logger:      log.New(io.Discard, "[Pool] ", 0),
		pingTimeout: pingTimeout,
	}
}

// Ping checks if a connection is likely alive.
// This is a basic check; a real ping might involve writing/reading data.
func (p *ConnectionPool) ping(conn *Client) bool {
	if conn == nil || conn.conn == nil { // Use conn.conn to access the underlying net.Conn
		return false
	}
	// Simple check: Set a short deadline and try a zero-byte read.
	// This often reveals closed connections without sending data.
	// Note: This isn't foolproof and might not work reliably across all OS/network conditions.
	err := conn.conn.SetReadDeadline(time.Now().Add(p.pingTimeout))
	if err != nil {
		p.logger.Printf("Ping: failed to set deadline for %s: %v", conn.RemoteAddr(), err)
		return false // Assume dead if can't set deadline
	}

	_, err = conn.conn.Read(make([]byte, 0)) // Attempt zero-byte read
	conn.conn.SetReadDeadline(time.Time{})   // Clear deadline immediately

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		// Timeout on zero-byte read usually means connection is alive (waiting for data)
		return true
	}
	if err == io.EOF || errors.Is(err, net.ErrClosed) {
		// EOF or explicitly closed means connection is dead
		p.logger.Printf("Ping: connection %s seems dead: %v", conn.RemoteAddr(), err)
		return false
	}
	// Other errors might indicate issues, treat as dead for safety
	if err != nil {
		p.logger.Printf("Ping: connection %s returned error: %v", conn.RemoteAddr(), err)
		return false
	}

	// No error on zero-byte read (might happen on some systems) - assume alive
	return true
}

func (p *ConnectionPool) Get() (*Client, error) {
	select {
	case conn := <-p.pool:
		// Check if the connection is still alive before returning
		if !p.ping(conn) {
			p.logger.Printf("Connection from pool failed ping, closing and creating new.")
			_ = conn.Close() // Close the dead connection
			// Try creating a new one instead of recursively calling Get()
			return p.factory()
		}
		p.logger.Printf("Reusing connection from pool: %s", conn.RemoteAddr())
		return conn, nil
	default:
		// Pool is empty, create a new connection
		p.logger.Printf("Pool empty, creating new connection.")
		return p.factory()
	}
}

func (p *ConnectionPool) Put(conn *Client) {
	if conn == nil {
		return
	}

	// Optional: Check connection health before putting back?
	// if !p.ping(conn) {
	//    p.logger.Printf("Connection failed ping before returning to pool, closing.")
	//    _ = conn.Close()
	//    return
	// }

	select {
	case p.pool <- conn:
		p.logger.Printf("Connection %s returned to pool", conn.RemoteAddr())
	default:
		// Pool is full, close the connection
		p.logger.Printf("Pool full, closing connection %s", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			p.logger.Printf("Error closing connection %s: %v", conn.RemoteAddr(), err)
		}
	}
}

// Close closes all connections in the pool and empties it.
func (p *ConnectionPool) Close() {
	p.logger.Printf("Closing connection pool...")
	close(p.pool) // Close the channel to prevent new puts

	// Drain the channel and close connections
	for conn := range p.pool {
		if err := conn.Close(); err != nil {
			p.logger.Printf("Error closing pooled connection %s: %v", conn.RemoteAddr(), err)
		}
	}
	p.logger.Printf("Connection pool closed.")
}
