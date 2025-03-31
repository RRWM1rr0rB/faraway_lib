// Package db provides database-optimized UUID generation with crypto-security.
package db

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// UUID represents a 128-bit UUID (v4 or v7) for database usage.
type UUID [16]byte

// NewV4 generates a cryptographically secure UUIDv4.
func NewV4() (UUID, error) {
	var u UUID
	if _, err := rand.Read(u[:]); err != nil {
		return UUID{}, fmt.Errorf("db: failed to generate v4: %w", err)
	}
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant 10xx
	return u, nil
}

// NewV7 generates a time-ordered UUIDv7 with crypto-secure entropy.
func NewV7() (UUID, error) {
	var u UUID
	now := time.Now().UnixMilli()

	// 48-bit timestamp (big-endian)
	binary.BigEndian.PutUint64(u[0:8], uint64(now)<<16)

	// Version 7 and variant bits
	u[6] = 0x70
	u[8] = 0x80

	// 62 random bits
	if _, err := rand.Read(u[6:]); err != nil {
		return UUID{}, fmt.Errorf("db: failed to generate v7: %w", err)
	}

	return u, nil
}
