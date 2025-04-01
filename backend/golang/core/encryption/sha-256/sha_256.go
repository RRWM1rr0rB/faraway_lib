package sha_256

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/rand"
)

// ChallengeInfo represents the information needed for a proof-of-work challenge
type ChallengeInfo struct {
	RandomString          string // Random string to be hashed
	NumberLeadingZeros    int32  // Required number of leading zeros in the hash
	SolutionNumberSymbols int32  // Length of the solution string
}

const (
	// AllowedSymbols contains the characters that can be used in random strings
	AllowedSymbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// generateCryptoRandomString generates a random string using crypto/rand
// Returns the generated string and any error that occurred
func generateCryptoRandomString(length int32) (string, error) {
	bytes := make([]byte, length)

	if _, err := cryptorand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate crypto random string: %w", err)
	}

	for i := range bytes {
		bytes[i] = AllowedSymbols[int(bytes[i])%len(AllowedSymbols)]
	}
	return string(bytes), nil
}

// GenerateMathRandomString generates a random string using math/rand
// This is a fallback method when crypto/rand is not available
func GenerateMathRandomString(length int32) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = AllowedSymbols[rand.Intn(len(AllowedSymbols))]
	}
	return string(result)
}

// IsHashValid checks if the hash of the given data has at least the required number of leading zeros
func IsHashValid(data string, requiredZeros int32) bool {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
	return CalculateLeadingZeros(hash) >= requiredZeros
}

// CalculateLeadingZeros counts the number of leading zeros in a hexadecimal hash string
// This function is exported for testing purposes
func CalculateLeadingZeros(hash string) int32 {
	var zeros int32
	for _, c := range hash {
		switch {
		case c == '0':
			zeros += 4
		case c == '1':
			zeros += 3
		case c >= '2' && c <= '3':
			zeros += 2
		case c >= '4' && c <= '7':
			zeros += 1
		default:
			return zeros
		}
	}
	return zeros
}
