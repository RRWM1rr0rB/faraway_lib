// tcp/ratelimiter.go
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	defaultRateLimitPerIP    = 10              // Max connection per sec.
	defaultInitialDifficulty = 4               // Start difficulty PoW(0000).
	maxDifficulty            = 8               // Max Difficulty PoW.
	banDuration              = 1 * time.Minute // Ban Duration (reduced for testing/demo).
	cleanupInterval          = 5 * time.Minute // Interval for cleaning up old entries.
)

type RateLimiter struct {
	mu sync.RWMutex
	// IP -> amount connection in this is second.
	connectionsPerIP map[string]*rateCounter
	// IP -> ban end time.
	bannedIPs map[string]time.Time
	// IP -> currently difficulty PoW.
	difficulties map[string]int32
	logger       *log.Logger

	// For background cleanup
	stopCh chan struct{}
	wg     sync.WaitGroup
}

type rateCounter struct {
	count     int64
	timestamp time.Time
}

// NewRateLimiter create new RateLimiter and starts background cleanup.
func NewRateLimiter(logger *log.Logger) *RateLimiter {
	if logger == nil {
		logger = log.New(io.Discard, "[RateLimiter] ", log.LstdFlags) // Use io.Discard or provide a default logger
	}
	rl := &RateLimiter{
		connectionsPerIP: make(map[string]*rateCounter),
		bannedIPs:        make(map[string]time.Time),
		difficulties:     make(map[string]int32),
		logger:           logger,
		stopCh:           make(chan struct{}),
	}

	rl.wg.Add(1)
	go rl.cleanupLoop()

	return rl
}

// Stop stops the background cleanup goroutine.
func (r *RateLimiter) Stop() {
	close(r.stopCh)
	r.wg.Wait()
	r.logger.Println("Cleanup routine stopped.")
}

func (r *RateLimiter) cleanupLoop() {
	defer r.wg.Done()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	r.logger.Printf("Starting cleanup routine with interval %v", cleanupInterval)
	for {
		select {
		case <-ticker.C:
			r.logger.Println("Running periodic cleanup...")
			r.Cleanup()
		case <-r.stopCh:
			r.logger.Println("Stopping cleanup routine...")
			return
		}
	}
}

// RateLimitMiddleware creates a middleware for limiting connections
// This middleware checks bans, rate limits, and performs PoW challenges.
// It CLOSES the connection if any check fails.
func RateLimitMiddleware(limiter *RateLimiter) func(conn net.Conn) bool {
	return func(conn net.Conn) bool { // Return bool indicating if connection should proceed
		addr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			limiter.logger.Printf("Could not get TCP address from connection")
			conn.Close() // Close invalid connection type
			return false
		}
		ip := addr.IP.String()

		// 1. Check if the IP is currently banned
		if limiter.isBanned(ip) {
			limiter.logger.Printf("IP %s rejected: currently banned", ip)
			conn.Close()
			return false
		}

		// 2. Check rate limit and potentially ban
		proceed, ban := limiter.checkAndUpdateRate(ip)
		if ban {
			limiter.logger.Printf("IP %s banned due to exceeding rate limit", ip)
			conn.Close()
			return false
		}
		if !proceed {
			// Rate limit exceeded, but not banned yet (e.g., requires PoW)
			limiter.logger.Printf("Rate limit exceeded for IP %s, requiring PoW", ip)

			// --- PoW Challenge ---
			// Increase difficulty *before* sending challenge
			limiter.increaseDifficulty(ip)
			difficulty := limiter.getDifficulty(ip) // Get potentially increased difficulty

			challenge, err := GeneratePoWChallenge(difficulty)
			if err != nil {
				limiter.logger.Printf("IP %s: Failed to generate PoW challenge: %v", ip, err)
				conn.Close()
				return false
			}

			// Send the challenge
			// Add a deadline for writing the challenge
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second)) // Example deadline
			if writeErr := WritePoWChallenge(conn, challenge); writeErr != nil {
				limiter.logger.Printf("IP %s: Failed to write PoW challenge: %v", ip, writeErr)
				conn.Close()
				conn.SetWriteDeadline(time.Time{}) // Clear deadline
				return false
			}
			conn.SetWriteDeadline(time.Time{}) // Clear deadline

			// Read the solution
			// Add a deadline for reading the solution
			conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // Example deadline
			solution, solutionErr := ReadPoWSolution(conn)
			if solutionErr != nil {
				// Differentiate between timeout and other errors
				if errors.Is(solutionErr, context.DeadlineExceeded) || (solutionErr != nil && solutionErr.Error() == "failed to read nonce: EOF") { // Check specific errors
					limiter.logger.Printf("IP %s: Did not receive PoW solution in time or connection closed", ip)
				} else {
					limiter.logger.Printf("IP %s: Failed to read PoW solution: %v", ip, solutionErr)
				}
				conn.Close()
				conn.SetReadDeadline(time.Time{}) // Clear deadline
				return false
			}
			conn.SetReadDeadline(time.Time{}) // Clear deadline

			// Validate the solution
			if !ValidatePoWSolution(challenge, solution) {
				limiter.logger.Printf("IP %s: Invalid PoW solution received", ip)
				// Difficulty was already increased
				conn.Close()
				return false
			}

			// If the solution is valid, decrease the difficulty slightly (optional, could just keep it high)
			// limiter.decreaseDifficulty(ip) // Optional: give benefit for solving hard puzzle
			limiter.logger.Printf("IP %s: Valid PoW solution received (difficulty %d)", ip, difficulty)
			// PoW passed, allow connection to proceed
			return true

		}

		// 3. If rate limit is okay, proceed directly without PoW
		limiter.logger.Printf("IP %s accepted (within rate limit)", ip)
		return true
	}
}

// Helper function to apply middleware in server handleConnection
func ApplyMiddleware(conn net.Conn, middleware func(net.Conn) bool, handler func(net.Conn)) {
	if middleware != nil {
		if !middleware(conn) {
			// Middleware rejected the connection and closed it
			return
		}
		// Middleware passed, connection is still open
	}
	// Proceed with the actual handler
	handler(conn)
}

func (r *RateLimiter) isBanned(ip string) bool {
	r.mu.RLock()
	banTime, exists := r.bannedIPs[ip]
	r.mu.RUnlock() // Unlock before potentially acquiring write lock

	if !exists {
		return false
	}

	if time.Now().After(banTime) {
		// Ban expired, remove it
		r.mu.Lock()
		// Double check after acquiring write lock
		if currentBanTime, stillExists := r.bannedIPs[ip]; stillExists && time.Now().After(currentBanTime) {
			delete(r.bannedIPs, ip)
			r.logger.Printf("Ban expired for IP %s", ip)
		}
		r.mu.Unlock()
		return false // Not banned anymore
	}

	// Ban is still active
	return true
}

// checkAndUpdateRate checks connection rate, updates counter, and returns (allowConnection, shouldBan)
func (r *RateLimiter) checkAndUpdateRate(ip string) (bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	counter, exists := r.connectionsPerIP[ip]

	// First connection within the second or first ever
	if !exists || now.Sub(counter.timestamp) >= time.Second {
		r.connectionsPerIP[ip] = &rateCounter{
			count:     1,
			timestamp: now,
		}
		// Reset difficulty if it was previously high. Optional.
		delete(r.difficulties, ip)
		return true, false // Allow, don't ban
	}

	// Connection within the same second
	counter.count++

	// Check if limit exceeded
	if counter.count > defaultRateLimitPerIP {
		// Ban the IP
		r.bannedIPs[ip] = now.Add(banDuration)
		// Remove the counter as the IP is now banned
		delete(r.connectionsPerIP, ip)
		return false, true // Don't allow, Ban
	}

	// Rate limit is okay for now
	return true, false // Allow, don't ban
}

// increaseDifficulty increases PoW difficulty for an IP.
// MUST be called with lock held or ensure thread safety.
func (r *RateLimiter) increaseDifficulty(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	currentDifficulty := r.getDifficultyLocked(ip) // Use locked version
	if currentDifficulty < maxDifficulty {
		newDifficulty := currentDifficulty + 1
		r.difficulties[ip] = newDifficulty
		r.logger.Printf("Increased PoW difficulty for IP %s to %d", ip, newDifficulty)
	}
}

// decreaseDifficulty decreases PoW difficulty for an IP.
// MUST be called with lock held or ensure thread safety.
func (r *RateLimiter) decreaseDifficulty(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	currentDifficulty := r.getDifficultyLocked(ip) // Use locked version
	if currentDifficulty > defaultInitialDifficulty {
		newDifficulty := currentDifficulty - 1
		r.difficulties[ip] = newDifficulty
		r.logger.Printf("Decreased PoW difficulty for IP %s to %d", ip, newDifficulty)
	} else if currentDifficulty == defaultInitialDifficulty {
		// Optionally remove the entry if it's back to default
		delete(r.difficulties, ip)
		r.logger.Printf("Reset PoW difficulty for IP %s to default", ip)
	}
}

// getDifficulty returns the current PoW difficulty for an IP.
func (r *RateLimiter) getDifficulty(ip string) int32 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getDifficultyLocked(ip)
}

// getDifficultyLocked returns the current PoW difficulty, assumes lock is already held.
func (r *RateLimiter) getDifficultyLocked(ip string) int32 {
	difficulty, exists := r.difficulties[ip]
	if !exists {
		// Set default difficulty if not exists, maybe?
		// r.difficulties[ip] = defaultInitialDifficulty
		return defaultInitialDifficulty
	}
	return difficulty
}

// Cleanup removes expired bans and old connection counters.
func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cleanedBans := 0
	cleanedCounters := 0

	// Clean up expired bans
	for ip, banTime := range r.bannedIPs {
		if now.After(banTime) {
			delete(r.bannedIPs, ip)
			cleanedBans++
		}
	}

	// Clean up old counters (older than 1-2 seconds)
	cutoff := now.Add(-2 * time.Second) // Keep counters for 2 seconds for safety
	for ip, counter := range r.connectionsPerIP {
		if counter.timestamp.Before(cutoff) {
			delete(r.connectionsPerIP, ip)
			cleanedCounters++
		}
	}

	// Clean up difficulties for IPs no longer tracked (optional)
	cleanedDifficulties := 0
	for ip := range r.difficulties {
		_, counterExists := r.connectionsPerIP[ip]
		_, banExists := r.bannedIPs[ip]
		if !counterExists && !banExists {
			// Only remove difficulty if IP is neither in counters nor banned
			delete(r.difficulties, ip)
			cleanedDifficulties++
		}
	}

	if cleanedBans > 0 || cleanedCounters > 0 || cleanedDifficulties > 0 {
		r.logger.Printf("Cleanup finished. Removed: %d bans, %d counters, %d difficulties.", cleanedBans, cleanedCounters, cleanedDifficulties)
	}
}
