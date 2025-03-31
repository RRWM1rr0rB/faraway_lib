package tcp

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"time"
)

type Server struct {
	address      string
	listener     net.Listener
	handler      func(net.Conn)
	logger       *log.Logger
	idleTimeout  time.Duration
	tlsConfig    *tls.Config
	shutdownChan chan struct{}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return wrapError("start server", err, false)
	}

	if s.tlsConfig != nil {
		listener = tls.NewListener(listener, s.tlsConfig)
	}

	s.listener = listener
	s.shutdownChan = make(chan struct{})

	go s.acceptConnections()
	s.logger.Printf("Server started on %s", s.address)
	return nil
}

func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					s.logger.Printf("Accept error: %v", err)
				}
				return
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(s.idleTimeout))
	s.handler(conn)
}

func (s *Server) Stop() error {
	close(s.shutdownChan)
	if err := s.listener.Close(); err != nil {
		return wrapError("stop server", err, false)
	}
	s.logger.Printf("Server stopped")
	return nil
}

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
