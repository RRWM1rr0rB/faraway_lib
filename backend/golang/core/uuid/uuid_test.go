package uuid_test

import (
	"testing"

	"trade/app/pkg/core/uuid/db"
	"trade/app/pkg/core/uuid/network"
)

func BenchmarkDBv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		db.NewV4()
	}
}

func BenchmarkDBv7(b *testing.B) {
	for i := 0; i < b.N; i++ {
		db.NewV7()
	}
}

func BenchmarkNetworkv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		network.NewV4()
	}
}

func BenchmarkNetworkv7(b *testing.B) {
	for i := 0; i < b.N; i++ {
		network.NewV7()
	}
}

func TestUUIDCompliance(t *testing.T) {
	// Test DB UUIDv4
	if _, err := db.NewV4(); err != nil {
		t.Errorf("DB UUIDv4 failed: %v", err)
	}

	// Test Network UUIDv7
	id := network.NewV7()
	if id[6]&0xf0 != 0x70 {
		t.Error("Network UUIDv7 version mismatch")
	}
}
