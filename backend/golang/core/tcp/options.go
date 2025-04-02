package tcp

import (
	"crypto/tls"
	"log"
	"net"
	"time"
)

// ClientOption defines functional options for configuring the Client.
type ClientOption func(*Client)

// ServerOption defines functional options for configuring the Server.
type ServerOption func(*Server)

// WithTimeouts sets the read and write timeouts for the Client.
func WithTimeouts(read, write time.Duration) ClientOption {
	return func(c *Client) {
		c.readTimeout = read
		c.writeTimeout = write
	}
}

// WithBufferSize sets the buffer size for the Client.
func WithBufferSize(size int) ClientOption {
	return func(c *Client) {
		c.bufferSize = size
	}
}

// WithClientLogger sets the logger for the Client.
func WithClientLogger(logger *log.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithTLSClientConfig sets the TLS configuration for the Client.
func WithTLSClientConfig(config *tls.Config) ClientOption {
	return func(c *Client) {
		c.tlsConfig = config
	}
}

// WithServerTimeout sets the idle timeout for the Server.
func WithServerTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.idleTimeout = timeout
	}
}

// WithServerLogger sets the logger for the Server.
func WithServerLogger(logger *log.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithServerTLS sets the TLS configuration for the Server.
func WithServerTLS(config *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConfig = config
	}
}

// WithPoolLogger Option to set logger for the pool
func WithPoolLogger(logger *log.Logger) func(*ConnectionPool) {
	return func(p *ConnectionPool) {
		p.logger = logger
	}
}

// WithPoolPingTimeout Option to set ping timeout for the pool
func WithPoolPingTimeout(timeout time.Duration) func(*ConnectionPool) {
	return func(p *ConnectionPool) {
		p.pingTimeout = timeout
	}
}

// WithMiddleware sets the middleware function for the Server.
func WithMiddleware(mw func(net.Conn) bool) ServerOption {
	return func(s *Server) {
		s.middleware = mw
	}
}
