package tcp

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"time"
)

type ConnectionStats struct {
	BytesRead    uint64
	BytesWritten uint64
	LastActivity time.Time
	RetryCount   int
}
type Client struct {
	address      string
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	bufferSize   int
	logger       *log.Logger
	tlsConfig    *tls.Config
	stats        ConnectionStats
}

func (c *Client) Connect() error {
	var conn net.Conn
	var err error

	if c.tlsConfig != nil {
		conn, err = tls.Dial("tcp", c.address, c.tlsConfig)
	} else {
		conn, err = net.Dial("tcp", c.address)
	}

	if err != nil {
		return wrapError("connect", err, true)
	}

	c.conn = conn
	c.stats = ConnectionStats{
		LastActivity: time.Now(),
	}
	c.logger.Printf("Connected to %s", c.address)
	return nil
}

func (c *Client) Read() ([]byte, error) {
	if c.conn == nil {
		return nil, &ConnectionError{Op: "read", Err: ErrConnectionClosed}
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		return nil, wrapError("set read deadline", err, false)
	}
	defer c.conn.SetReadDeadline(time.Time{})

	buf := make([]byte, c.bufferSize)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, wrapError("read", err, isNetworkErrorRetryable(err))
	}

	c.stats.BytesRead += uint64(n)
	c.stats.LastActivity = time.Now()
	return buf[:n], nil
}

func (c *Client) Write(data []byte) error {
	if c.conn == nil {
		return &ConnectionError{Op: "write", Err: ErrConnectionClosed}
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return wrapError("set write deadline", err, false)
	}
	defer c.conn.SetWriteDeadline(time.Time{})

	n, err := c.conn.Write(data)
	if err != nil {
		return wrapError("write", err, isNetworkErrorRetryable(err))
	}

	c.stats.BytesWritten += uint64(n)
	c.stats.LastActivity = time.Now()
	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	if err := c.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		c.logger.Printf("Close error: %v", err)
		return wrapError("close", err, false)
	}

	c.logger.Printf("Connection closed")
	c.conn = nil
	return nil
}

func (c *Client) Reconnect() error {
	c.Close()
	return c.Connect()
}

func (c *Client) RemoteAddr() net.Addr {
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

func (c *Client) Stats() ConnectionStats {
	return c.stats
}
