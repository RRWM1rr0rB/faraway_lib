package tcp

import (
	"log"
	"net"
	"sync"
	"time"
)

const (
	defaultRateLimitPerIP    = 10             // максимальное количество соединений в секунду с одного IP
	defaultInitialDifficulty = 4              // начальная сложность PoW (количество нулей)
	maxDifficulty            = 8              // максимальная сложность PoW
	banDuration              = 24 * time.Hour // длительность бана
)

// RateLimiter представляет собой структуру для отслеживания соединений
type RateLimiter struct {
	mu sync.RWMutex
	// IP -> количество соединений в текущей секунде
	connectionsPerIP map[string]*rateCounter
	// IP -> время окончания бана
	bannedIPs map[string]time.Time
	// IP -> текущая сложность PoW
	difficulties map[string]int32
	logger       *log.Logger
}

type rateCounter struct {
	count     int64
	timestamp time.Time
}

// NewRateLimiter создает новый RateLimiter
func NewRateLimiter(logger *log.Logger) *RateLimiter {
	if logger == nil {
		logger = log.Default()
	}
	return &RateLimiter{
		connectionsPerIP: make(map[string]*rateCounter),
		bannedIPs:        make(map[string]time.Time),
		difficulties:     make(map[string]int32),
		logger:           logger,
	}
}

// RateLimitMiddleware создает middleware для ограничения соединений
func RateLimitMiddleware(limiter *RateLimiter) func(net.Conn) {
	return func(conn net.Conn) {
		ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

		// Проверяем, не забанен ли IP
		if limiter.isBanned(ip) {
			limiter.logger.Printf("IP %s is banned", ip)
			conn.Close()
			return
		}

		// Проверяем количество соединений
		if !limiter.allowConnection(ip) {
			limiter.logger.Printf("Rate limit exceeded for IP %s", ip)
			// Увеличиваем сложность PoW
			limiter.increaseDifficulty(ip)
			conn.Close()
			return
		}

		// Получаем текущую сложность для IP
		difficulty := limiter.getDifficulty(ip)

		// Генерируем PoW-задачу
		challenge, err := GeneratePoWChallenge(difficulty)
		if err != nil {
			limiter.logger.Printf("Failed to generate PoW challenge: %v", err)
			conn.Close()
			return
		}

		// Отправляем задачу клиенту
		if err := WritePoWChallenge(conn, challenge); err != nil {
			limiter.logger.Printf("Failed to write PoW challenge: %v", err)
			conn.Close()
			return
		}

		// Читаем решение от клиента
		solution, err := ReadPoWSolution(conn)
		if err != nil {
			limiter.logger.Printf("Failed to read PoW solution: %v", err)
			conn.Close()
			return
		}

		// Проверяем решение
		if !ValidatePoWSolution(challenge, solution) {
			limiter.logger.Printf("Invalid PoW solution from IP %s", ip)
			limiter.increaseDifficulty(ip)
			conn.Close()
			return
		}

		// Если решение валидно, уменьшаем сложность
		limiter.decreaseDifficulty(ip)
		limiter.logger.Printf("Valid PoW solution from IP %s, difficulty: %d", ip, difficulty)
	}
}

// isBanned проверяет, забанен ли IP
func (r *RateLimiter) isBanned(ip string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	banTime, exists := r.bannedIPs[ip]
	if !exists {
		return false
	}

	if time.Now().After(banTime) {
		// Удаляем истекший бан
		r.mu.RUnlock()
		r.mu.Lock()
		delete(r.bannedIPs, ip)
		r.mu.Unlock()
		return false
	}

	return true
}

// allowConnection проверяет, разрешено ли новое соединение
func (r *RateLimiter) allowConnection(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	counter, exists := r.connectionsPerIP[ip]

	if !exists {
		r.connectionsPerIP[ip] = &rateCounter{
			count:     1,
			timestamp: now,
		}
		return true
	}

	// Если прошла секунда, сбрасываем счетчик
	if now.Sub(counter.timestamp) >= time.Second {
		counter.count = 1
		counter.timestamp = now
		return true
	}

	// Увеличиваем счетчик
	counter.count++

	// Если превышен лимит, бан IP
	if counter.count > defaultRateLimitPerIP {
		r.bannedIPs[ip] = now.Add(banDuration)
		return false
	}

	return true
}

// increaseDifficulty увеличивает сложность PoW для IP
func (r *RateLimiter) increaseDifficulty(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.difficulties[ip]
	if current < maxDifficulty {
		r.difficulties[ip] = current + 1
		r.logger.Printf("Increased PoW difficulty for IP %s to %d", ip, current+1)
	}
}

// decreaseDifficulty уменьшает сложность PoW для IP
func (r *RateLimiter) decreaseDifficulty(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.difficulties[ip]
	if current > defaultInitialDifficulty {
		r.difficulties[ip] = current - 1
		r.logger.Printf("Decreased PoW difficulty for IP %s to %d", ip, current-1)
	}
}

// getDifficulty возвращает текущую сложность PoW для IP
func (r *RateLimiter) getDifficulty(ip string) int32 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	difficulty, exists := r.difficulties[ip]
	if !exists {
		return defaultInitialDifficulty
	}
	return difficulty
}

// Cleanup удаляет устаревшие записи
func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Очищаем истекшие баны
	for ip, banTime := range r.bannedIPs {
		if now.After(banTime) {
			delete(r.bannedIPs, ip)
		}
	}

	// Очищаем старые счетчики
	for ip, counter := range r.connectionsPerIP {
		if now.Sub(counter.timestamp) >= time.Second {
			delete(r.connectionsPerIP, ip)
		}
	}
}
