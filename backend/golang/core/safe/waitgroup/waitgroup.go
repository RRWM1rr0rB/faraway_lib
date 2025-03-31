// Package waitgroup provides a thread-safe WaitGroup with safety checks.
package waitgroup

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// WaitGroup enhances sync.WaitGroup with atomic counters and panic on misuse.
type WaitGroup struct {
	wg     sync.WaitGroup
	mu     sync.RWMutex
	count  int32
	panics bool
}

// Option configures WaitGroup behavior.
type Option func(*WaitGroup)

// WithPanicOnMisuse enables panic on negative counter.
func WithPanicOnMisuse() Option {
	return func(wg *WaitGroup) {
		wg.panics = true
	}
}

// NewWaitGroup creates a configured WaitGroup.
func NewWaitGroup(opts ...Option) *WaitGroup {
	wg := &WaitGroup{}
	for _, opt := range opts {
		opt(wg)
	}
	return wg
}

// Add increments the counter. Returns error or panics on negative value.
func (wg *WaitGroup) Add(delta int) error {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	newCount := atomic.AddInt32(&wg.count, int32(delta))
	if newCount < 0 {
		err := fmt.Errorf("safe: waitgroup counter went negative (%d)", newCount)
		if wg.panics {
			panic(err)
		}
		return err
	}
	wg.wg.Add(delta)
	return nil
}

// Done decrements the counter. Returns error or panics on negative value.
func (wg *WaitGroup) Done() error {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	newCount := atomic.AddInt32(&wg.count, -1)
	if newCount < 0 {
		err := fmt.Errorf("safe: waitgroup counter went negative (%d)", newCount)
		if wg.panics {
			panic(err)
		}
		return err
	}
	wg.wg.Done()
	return nil
}

// Wait blocks until the counter reaches zero.
func (wg *WaitGroup) Wait() {
	wg.mu.RLock()
	defer wg.mu.RUnlock()
	wg.wg.Wait()
}

// Count returns the current counter value (thread-safe).
func (wg *WaitGroup) Count() int32 {
	return atomic.LoadInt32(&wg.count)
}
