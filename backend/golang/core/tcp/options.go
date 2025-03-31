package tcp

import (
	"crypto/tls"
	"log"
	"time"
)

type TCPClientOption func(*TCPClient)
type ServerOption func(*TCPServer)

func WithTimeouts(read, write time.Duration) TCPClientOption {
	return func(c *TCPClient) {
		c.readTimeout = read
		c.writeTimeout = write
	}
}

func WithBufferSize(size int) TCPClientOption {
	return func(c *TCPClient) {
		c.bufferSize = size
	}
}

func WithClientLogger(logger *log.Logger) TCPClientOption {
	return func(c *TCPClient) {
		c.logger = logger
	}
}

func WithTLSClientConfig(config *tls.Config) TCPClientOption {
	return func(c *TCPClient) {
		c.tlsConfig = config
	}
}

func WithServerTimeout(timeout time.Duration) ServerOption {
	return func(s *TCPServer) {
		s.idleTimeout = timeout
	}
}

func WithServerLogger(logger *log.Logger) ServerOption {
	return func(s *TCPServer) {
		s.logger = logger
	}
}

func WithServerTLS(config *tls.Config) ServerOption {
	return func(s *TCPServer) {
		s.tlsConfig = config
	}
}
