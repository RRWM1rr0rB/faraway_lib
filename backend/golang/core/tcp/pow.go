package tcp

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// PoWChallenge представляет собой структуру для PoW-задачи
type PoWChallenge struct {
	Timestamp   int64
	RandomBytes []byte
	Difficulty  int32
}

// PoWSolution представляет собой решение PoW-задачи
type PoWSolution struct {
	Nonce uint64
}

// GeneratePoWChallenge создает новую PoW-задачу
func GeneratePoWChallenge(difficulty int32) (*PoWChallenge, error) {
	if difficulty < 0 || difficulty > 256 {
		return nil, fmt.Errorf("invalid difficulty")
	}

	randomBytes := make([]byte, 32)

	if _, err := cryptorand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return &PoWChallenge{
		Timestamp:   time.Now().Unix(),
		RandomBytes: randomBytes,
		Difficulty:  difficulty,
	}, nil
}

// ValidatePoWSolution проверяет решение PoW-задачи
func ValidatePoWSolution(challenge *PoWChallenge, solution *PoWSolution) bool {
	if time.Now().Unix()-challenge.Timestamp > 60 {
		return false
	}
	// Создаем буфер для хеширования
	buf := make([]byte, 8+32+8)
	binary.BigEndian.PutUint64(buf[0:8], uint64(challenge.Timestamp))
	copy(buf[8:40], challenge.RandomBytes)
	binary.BigEndian.PutUint64(buf[40:48], solution.Nonce)

	// Вычисляем хеш
	hash := sha256.Sum256(buf)

	// Проверяем количество ведущих нулей
	leadingZeros := countLeadingZeros(hash[:])
	return leadingZeros >= challenge.Difficulty
}

// countLeadingZeros подсчитывает количество ведущих нулевых битов в байтовом массиве
func countLeadingZeros(data []byte) int32 {
	var zeros int32
	for _, b := range data {
		if b == 0 {
			zeros += 8
		} else {
			// Подсчитываем нулевые биты в текущем байте
			for i := 7; i >= 0; i-- {
				if (b >> i) == 0 {
					zeros++
				} else {
					return zeros
				}
			}
		}
	}
	return zeros
}

// WritePoWChallenge записывает PoW-задачу в writer
func WritePoWChallenge(w io.Writer, challenge *PoWChallenge) error {
	// Записываем timestamp (8 байт)
	if err := binary.Write(w, binary.BigEndian, challenge.Timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}

	// Записываем random bytes (32 байта)
	if _, err := w.Write(challenge.RandomBytes); err != nil {
		return fmt.Errorf("failed to write random bytes: %w", err)
	}

	// Записываем difficulty (4 байта)
	if err := binary.Write(w, binary.BigEndian, challenge.Difficulty); err != nil {
		return fmt.Errorf("failed to write difficulty: %w", err)
	}

	return nil
}

// ReadPoWChallenge читает PoW-задачу из reader
func ReadPoWChallenge(r io.Reader) (*PoWChallenge, error) {
	challenge := &PoWChallenge{
		RandomBytes: make([]byte, 32),
	}

	// Читаем timestamp (8 байт)
	if err := binary.Read(r, binary.BigEndian, &challenge.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}

	// Читаем random bytes (32 байта)
	if _, err := io.ReadFull(r, challenge.RandomBytes); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	// Читаем difficulty (4 байта)
	if err := binary.Read(r, binary.BigEndian, &challenge.Difficulty); err != nil {
		return nil, fmt.Errorf("failed to read difficulty: %w", err)
	}

	return challenge, nil
}

// WritePoWSolution записывает решение PoW-задачи в writer
func WritePoWSolution(w io.Writer, solution *PoWSolution) error {
	return binary.Write(w, binary.BigEndian, solution.Nonce)
}

// ReadPoWSolution читает решение PoW-задачи из reader
func ReadPoWSolution(r io.Reader) (*PoWSolution, error) {
	var solution PoWSolution
	if err := binary.Read(r, binary.BigEndian, &solution.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}
	return &solution, nil
}
