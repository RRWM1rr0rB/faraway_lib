package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultIdleTimeout = 5 * time.Minute
)

// ServerStats represents statistics about the server
type ServerStats struct {
	ActiveConnections int64
	TotalConnections  int64
	BytesRead         int64
	BytesWritten      int64
	LastActivity      time.Time
}

// Server represents a TCP server with connection management and statistics
type Server struct {
	address      string
	listener     net.Listener
	handler      func(net.Conn)
	logger       *log.Logger
	idleTimeout  time.Duration
	tlsConfig    *tls.Config
	ctx          context.Context
	cancel       context.CancelFunc
	stats        ServerStats
	mu           sync.RWMutex
	wg           sync.WaitGroup
	maxConns     int64
	currentConns int64
	middleware   func(net.Conn) bool
}

// NewServer creates a new TCP server with the given configuration
func NewServer(
	address string,
	handler func(net.Conn),
	tlsConfig *tls.Config,
	opts ...ServerOption,
) (*Server, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		address:     address,
		handler:     handler,
		tlsConfig:   tlsConfig,
		idleTimeout: defaultIdleTimeout,
		logger:      log.Default(),
		ctx:         ctx,
		cancel:      cancel,
		maxConns:    65101, // default max connections
		stats: ServerStats{
			LastActivity: time.Now(),
		},
	}

	for _, opt := range opts {
		opt(server)
	}

	return server, nil
}

// Start starts the server and begins accepting connections
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return errors.New("server already started")
	}

	listener, err := net.Listen(TCP, s.address)
	if err != nil {
		return wrapError("start server", err, false)
	}

	if s.tlsConfig != nil {
		listener = tls.NewListener(listener, s.tlsConfig)
	}

	s.listener = listener
	s.stats.LastActivity = time.Now()

	go s.acceptConnections()
	s.logger.Printf("Server started on %s", s.address)
	return nil
}

// acceptConnections accepts incoming connections and handles them
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			// Server is stopping
			if s.listener != nil {
				if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
					s.logger.Printf("Error closing listener: %v", err)
				}
			}
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					s.logger.Printf("Accept error: %v", err)
				}
				// If the server context is cancelled, listener might be closed, so we return.
				select {
				case <-s.ctx.Done():
					return
				default:
					// Continue accepting if it's a temporary error.
					continue
				}
			}

			if atomic.LoadInt64(&s.currentConns) >= s.maxConns {
				s.logger.Printf("Max connections reached, rejecting connection from %s", conn.RemoteAddr())
				conn.Close()
				continue
			}

			atomic.AddInt64(&s.currentConns, 1)
			atomic.AddInt64(&s.stats.TotalConnections, 1)
			atomic.AddInt64(&s.stats.ActiveConnections, 1)

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	addr := conn.RemoteAddr()
	s.logger.Printf("Connection from %s (%s)", addr, addr.Network())

	defer func() {
		atomic.AddInt64(&s.currentConns, -1)
		atomic.AddInt64(&s.stats.ActiveConnections, -1)
		// Ensure connection is closed on exit, check error
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			s.logger.Printf("Error closing connection from %s in defer: %v", addr, err)
		}
		s.wg.Done()
		s.logger.Printf("Connection closed: %s", addr) // Log connection closure
	}()

	if err := conn.SetDeadline(time.Now().Add(s.idleTimeout)); err != nil {
		s.logger.Printf("Set deadline error: %v", err)
		return
	}

	// Apply the middleware before handling the connection
	ApplyMiddleware(conn, s.middleware, func(passedConn net.Conn) {
		// If middleware passed, run the original handler
		// Ensure the handler also manages deadlines if necessary
		s.handler(passedConn)
	})
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener == nil {
		return errors.New("server not started")
	}

	s.cancel() // Signal goroutines to stop

	// Close the listener to stop accepting new connections
	if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return wrapError("stop server", err, false)
	}

	// Wait for all active connections to close
	s.wg.Wait()
	s.logger.Printf("Server stopped")
	return nil
}

// StopWithTimeout gracefully stops the server with a timeout
func (s *Server) StopWithTimeout(timeout time.Duration) error {
	deadline := time.After(timeout)
	done := make(chan error)

	go func() {
		done <- s.Stop()
	}()

	select {
	case <-deadline:
		return ErrTimeout
	case err := <-done:
		return err
	}
}

// Stats returns the current server statistics
func (s *Server) Stats() ServerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// SetMaxConnections sets the maximum number of concurrent connections
func (s *Server) SetMaxConnections(max int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxConns = max
}
