// Package google_uuid provides ULID generation with cryptographically secure entropy.
package google_uuid

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand/v2"

	"github.com/oklog/ulid/v2"
)

// ULIDGenerator implements ULID generation with ChaCha8 seeded by crypto/rand.
type ULIDGenerator struct {
	entropy *mathrand.ChaCha8
}

// chaChaReader adapts mathrand.ChaCha8 to io.Reader interface.
type chaChaReader struct {
	rng *mathrand.ChaCha8
}

// Read implements io.Reader for ChaCha8 (cryptographically secure).
func (c *chaChaReader) Read(p []byte) (n int, err error) {
	buffer := make([]byte, len(p))
	for i := 0; i < len(buffer); i += 8 {
		val := c.rng.Uint64()
		binary.LittleEndian.PutUint64(buffer[i:], val)
	}
	copy(p, buffer)
	return len(p), nil
}

// NewULIDGenerator creates a ULIDGenerator with a secure seed from crypto/rand.
func NewULIDGenerator() (*ULIDGenerator, error) {
	seed := make([]byte, 32)
	if _, err := cryptorand.Read(seed); err != nil {
		return nil, fmt.Errorf("failed to generate seed: %w", err)
	}

	var seedArr [32]byte
	copy(seedArr[:], seed)

	return &ULIDGenerator{
		entropy: mathrand.NewChaCha8(seedArr),
	}, nil
}

// GenerateID generates a ULID with cryptographically secure entropy.
func (g *ULIDGenerator) GenerateID() (string, error) {
	entropy := &chaChaReader{rng: g.entropy}
	id, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return id.String(), nil
}
