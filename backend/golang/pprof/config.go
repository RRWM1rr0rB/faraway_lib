package pprof

import (
	"time"
)

type Config struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
}

func NewConfig(host string, port int, readHeaderTimeout time.Duration) Config {
	return Config{Host: host, Port: port, ReadHeaderTimeout: readHeaderTimeout}
}
