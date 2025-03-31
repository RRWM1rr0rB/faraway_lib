package tracing

import (
	"errors"
)

var (
	ErrHostIsEmpty = errors.New("host cannot be empty")
	ErrPortIsEmpty = errors.New("port cannot be empty")
)

const (
	defaultHost = "localhost"
	defaultPort = "4318"
)

// Config holds tracing configuration parameters.
type config struct {
	host           string
	port           string
	serviceID      string
	serviceName    string
	serviceVersion string
	envName        string
}

// Validate checks required fields.
func (c *config) Validate() error {
	if c.host == "" {
		return ErrHostIsEmpty
	}
	if c.port == "" {
		return ErrPortIsEmpty
	}
	return nil
}

// ConfigParam configures the tracing setup.
type ConfigParam func(*config)

// WithHost sets the OTLP collector host.
func WithHost(host string) ConfigParam {
	return func(c *config) { c.host = host }
}

// WithPort sets the OTLP collector port.
func WithPort(port string) ConfigParam {
	return func(c *config) { c.port = port }
}

// WithServiceID sets the service instance ID.
func WithServiceID(id string) ConfigParam {
	return func(c *config) { c.serviceID = id }
}

// WithServiceName sets the service name.
func WithServiceName(name string) ConfigParam {
	return func(c *config) { c.serviceName = name }
}

// WithServiceVersion sets the service version.
func WithServiceVersion(version string) ConfigParam {
	return func(c *config) { c.serviceVersion = version }
}

// WithEnvName sets the deployment environment.
func WithEnvName(env string) ConfigParam {
	return func(c *config) { c.envName = env }
}
