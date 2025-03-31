// Package uuid provides RFC-compliant UUID generation using google/uuid.
package google_uuid

import (
	"github.com/google/uuid"
)

// GoogleUUIDGenerator implements IDGenerator for UUIDv4.
type GoogleUUIDGenerator struct{}

// NewGoogleUUIDGenerator creates a new UUIDv4 generator.
func NewGoogleUUIDGenerator() *GoogleUUIDGenerator {
	return &GoogleUUIDGenerator{}
}

// GenerateID produces a RFC 4122-compliant UUIDv4 string.
func (g *GoogleUUIDGenerator) GenerateID() string {
	return uuid.NewString()
}
