// Package uuid provides interfaces for ID generation.
package google_uuid

// IDGenerator defines the contract for ID generation implementations.
type IDGenerator interface {
	GenerateID() string
}
