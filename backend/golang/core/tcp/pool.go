package tcp

import (
	"io"
	"log"
)

type ConnectionPool struct {
	factory func() (*TCPClient, error)
	pool    chan *TCPClient
	maxSize int
	logger  *log.Logger
}

func NewConnectionPool(factory func() (*TCPClient, error), maxSize int) *ConnectionPool {
	return &ConnectionPool{
		factory: factory,
		pool:    make(chan *TCPClient, maxSize),
		maxSize: maxSize,
		logger:  log.New(io.Discard, "", 0),
	}
}

func (p *ConnectionPool) Get() (*TCPClient, error) {
	select {
	case conn := <-p.pool:
		return conn, nil
	default:
		return p.factory()
	}
}

func (p *ConnectionPool) Put(conn *TCPClient) {
	select {
	case p.pool <- conn:
		p.logger.Printf("Connection returned to pool")
	default:
		conn.Close()
		p.logger.Printf("Connection pool full, connection closed")
	}
}
