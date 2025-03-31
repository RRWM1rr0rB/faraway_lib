// error.go
package redis

import "fmt"

// Sorted Set errors
func ErrZAdd(err error) error {
	return fmt.Errorf("failed to ZAdd due to error: %w", err)
}

func ErrZRange(err error) error {
	return fmt.Errorf("failed to ZRange due to error: %w", err)
}

func ErrZRem(err error) error {
	return fmt.Errorf("failed to ZRem due to error: %w", err)
}

// List errors
func ErrLPush(err error) error {
	return fmt.Errorf("failed to LPush due to error: %w", err)
}

func ErrRPush(err error) error {
	return fmt.Errorf("failed to RPush due to error: %w", err)
}

func ErrLPop(err error) error {
	return fmt.Errorf("failed to LPop due to error: %w", err)
}

// Set errors
func ErrSAdd(err error) error {
	return fmt.Errorf("failed to SAdd due to error: %w", err)
}

// Generic command error
func ErrCmd(err error, cmd string) error {
	return fmt.Errorf("redis command %q failed: %w", cmd, err)
}
