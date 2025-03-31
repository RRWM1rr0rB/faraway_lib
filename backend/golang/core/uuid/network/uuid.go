// Package network provides high-speed UUID generation for network operations.
package network

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand/v2"
	"time"
)

// UUID represents a 128-bit UUID optimized for network performance.
type UUID [16]byte

var fastRand *mathrand.ChaCha8

func init() {
	// Initialize 32-byte seed with cryptographic randomness
	var seed [32]byte
	if _, err := cryptorand.Read(seed[:]); err != nil {
		panic(fmt.Sprintf("failed to initialize seed: %v", err))
	}

	// Inject timestamp for additional entropy
	binary.LittleEndian.PutUint64(seed[24:], uint64(time.Now().UnixNano()))

	fastRand = mathrand.NewChaCha8(seed)
}

// NewV4 generates a high-speed UUIDv4.
func NewV4() UUID {
	var u UUID
	binary.LittleEndian.PutUint64(u[0:8], fastRand.Uint64())
	binary.LittleEndian.PutUint64(u[8:16], fastRand.Uint64())
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant 10xx
	return u
}

// NewV7 generates a time-ordered UUIDv7 with millisecond precision.
func NewV7() UUID {
	var u UUID
	now := time.Now().UnixMilli()

	// 48-bit timestamp (big-endian)
	binary.BigEndian.PutUint64(u[0:8], uint64(now)<<16)

	// Version 7 (bits 49-52) and variant (bits 65-66)
	u[6] = 0x70
	u[8] = 0x80

	// 62 random bits using ChaCha8
	binary.LittleEndian.PutUint64(u[6:14], fastRand.Uint64())

	return u
}
