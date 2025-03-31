// Package uuid provides general-purpose UUID generation and handling.
package uuid

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand/v2"
	"time"
)

// UUID represents a 128-bit UUID (RFC 4122 and draft UUIDv7).
type UUID [16]byte

// NewV4 generates a RFC-compliant UUIDv4.
func NewV4() (UUID, error) {
	var u UUID
	if _, err := cryptorand.Read(u[:]); err != nil {
		return UUID{}, fmt.Errorf("uuid: v4 generation failed: %w", err)
	}
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant 10xx
	return u, nil
}

// NewV7 generates a time-ordered UUIDv7 with configurable source.
func NewV7(r *mathrand.ChaCha8) (UUID, error) {
	var u UUID
	now := time.Now().UnixMilli()
	binary.BigEndian.PutUint64(u[0:8], uint64(now)<<16)
	u[6] = 0x70 // Version 7
	u[8] = 0x80 // Variant 10xx

	if r == nil {
		var seed [32]byte
		if _, err := cryptorand.Read(seed[:]); err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
		r = mathrand.NewChaCha8(seed)
	}

	binary.LittleEndian.PutUint64(u[6:14], r.Uint64())

	return u, nil
}
