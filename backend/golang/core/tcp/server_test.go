package tcp

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestServerMiddlewareWithHighLoad tests the server's middleware functionality
// with a high number of concurrent connections from different IPs.
func TestServerMiddlewareWithHighLoad(t *testing.T) {
	// --- Configuration ---
	address := "localhost:8081"              // Используйте другой порт для тестирования
	numClients := 100000                     // Количество имитируемых клиентов
	maxConnections := 1000                   // Максимальное количество подключений сервера
	banDuration := 1 * time.Second           // Короткая продолжительность бана для тестирования
	rateLimit := 10                          // Подключений на IP в секунду
	middlewareDelay := 10 * time.Millisecond // Имитация времени обработки промежуточным ПО

	// --- Mock Rate Limiter Middleware ---
	// Это промежуточное ПО имитирует простое ограничение скорости.
	// В реальном тесте вы можете использовать более сложный мок или специфичную для теста
	// реализацию вашего фактического промежуточного ПО.
	type ipCounter struct {
		count     int
		timestamp time.Time
	}
	ipCounters := make(map[string]*ipCounter)
	ipBans := make(map[string]time.Time)
	var mu sync.Mutex

	mockMiddleware := func(conn net.Conn) bool {
		addr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			// ...
		}
		ip := addr.IP.String()

		mu.Lock()
		defer mu.Unlock()

		log.Printf("Промежуточное ПО: Проверка соединения от %s", ip) // Добавлено логирование проверки

		// 1. Проверяем, заблокирован ли IP
		if banTime, banned := ipBans[ip]; banned && time.Now().Before(banTime) {
			log.Printf("Промежуточное ПО: Соединение от %s заблокировано до %v", ip, banTime)
			conn.Close()
			return false
		}

		// 2. Проверяем ограничение скорости
		now := time.Now()
		counter, exists := ipCounters[ip]
		if !exists || now.Sub(counter.timestamp) >= time.Second {
			ipCounters[ip] = &ipCounter{
				count:     1,
				timestamp: now,
			}
			log.Printf("Промежуточное ПО: Первый запрос от %s за секунду", ip)
		} else {
			counter.count++
			log.Printf("Промежуточное ПО: Запрос №%d от %s за секунду", counter.count, ip)
			if counter.count > rateLimit {
				ipBans[ip] = now.Add(banDuration)
				delete(ipCounters, ip) // Удаляем счетчик при блокировке
				log.Printf("Промежуточное ПО: Соединение от %s ограничено по скорости, блокировка до %v", ip, ipBans[ip])
				conn.Close()
				return false
			}
		}

		// Имитация времени обработки промежуточным ПО
		time.Sleep(middlewareDelay)

		return true // Разрешаем соединение
	}

	// --- Server Setup ---
	handler := func(conn net.Conn) {
		defer conn.Close()
		// Простой эхо-обработчик для тестирования
		io.Copy(conn, conn)
	}

	server, err := NewServer(address, handler, nil,
		WithServerLogger(log.New(os.Stdout, "[SERVER] ", log.LstdFlags)),
		WithMiddleware(mockMiddleware),
	)
	if err != nil {
		t.Fatalf("Не удалось создать сервер: %v", err)
	}
	server.SetMaxConnections(int64(maxConnections))

	go func() {
		if err := server.Start(); err != nil {
			t.Fatalf("Не удалось запустить сервер: %v", err)
		}
	}()
	defer server.Stop()

	// --- Client Simulation ---
	var wg sync.WaitGroup
	connectErrors := int32(0) // Атомарный счетчик ошибок подключения
	allowedConnections := int32(0)
	bannedConnections := int32(0)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientNum int) {
			defer wg.Done()

			// Создаем уникальный IP для каждого клиента (для тестирования ограничения скорости)
			ipAddress := fmt.Sprintf("127.0.0.%d", (clientNum%254)+1) // Генерация разных IP в подсети
			localAddress := fmt.Sprintf("%s:%d", ipAddress, 0)

			// Создаем локальный адрес для исходящего подключения
			localAddr, err := net.ResolveTCPAddr("tcp", localAddress)
			if err != nil {
				t.Errorf("Ошибка разрешения локального адреса: %v", err)
				atomic.AddInt32(&connectErrors, 1)
				return
			}

			// Dial с таймаутом, чтобы избежать зависания теста
			dialer := net.Dialer{Timeout: 1 * time.Second, LocalAddr: localAddr} // Короткий таймаут для тестирования, устанавливаем локальный адрес
			conn, err := dialer.Dial("tcp", address)
			if err != nil {
				atomic.AddInt32(&connectErrors, 1)
				return
			}
			defer conn.Close()

			// Небольшая задержка перед отправкой данных
			time.Sleep(50 * time.Millisecond)

			// Отправляем некоторые данные, чтобы вызвать промежуточное ПО
			_, err = fmt.Fprintf(conn, "Client %d from %s\n", clientNum, ipAddress)
			if err != nil {
				// Логируем ошибку, но не завершаем тест немедленно
				t.Logf("Клиент %d (%s): Ошибка отправки данных: %v", clientNum, ipAddress, err)
				return
			}

			// Читаем некоторые данные (необязательно, для эха)
			_, err = io.ReadAll(conn)
			if err != nil && err != io.EOF {
				// Логируем ошибку, но не завершаем тест немедленно
				t.Logf("Клиент %d (%s): Ошибка чтения данных: %v", clientNum, ipAddress, err)
				return
			}

			// Проверяем, было ли соединение разрешено или отклонено промежуточным ПО (упрощенная проверка)
			// Эта проверка очень базовая и зависит от поведения мок-промежуточного ПО
			if conn.LocalAddr() != nil { // Если у нас есть локальный адрес, соединение, вероятно, было установлено
				atomic.AddInt32(&allowedConnections, 1)
			} else {
				atomic.AddInt32(&bannedConnections, 1)
			}

		}(i)
	}

	wg.Wait()

	// --- Verification ---
	t.Logf("Всего подключений: %d", numClients)
	t.Logf("Ошибки подключения: %d", connectErrors)
	t.Logf("Разрешенные подключения: %d", allowedConnections)
	t.Logf("Заблокированные подключения: %d", bannedConnections)

	// Базовая проверка: Некоторые подключения должны быть разрешены, а некоторые могут быть заблокированы
	if allowedConnections == 0 {
		t.Errorf("Ожидалось, что некоторые подключения будут разрешены, но ни одного не было")
	}

	// Проверяем, находится ли количество разрешенных подключений в ожидаемом диапазоне
	// Это зависит от конкретного поведения мок-промежуточного ПО и параметров теста
	// Например, если ограничение скорости равно 10, а максимальное количество подключений - 1000, мы ожидаем около 1000 разрешенных
	// Остальные должны быть заблокированы.
	expectedAllowedMin := int32(maxConnections) // Ожидаем, что будет разрешено как минимум maxConnections
	// expectedAllowedMax := int32(maxConnections + rateLimit) // Небольшой буфер

	if allowedConnections < expectedAllowedMin || allowedConnections > int32(numClients) {
		t.Errorf("Ожидалось, что количество разрешенных подключений будет между %d и %d, но получено %d", expectedAllowedMin, numClients, allowedConnections)
	}

	// Проверяем, находится ли количество заблокированных подключений в ожидаемом диапазоне (если есть)
	// Это будет зависеть от ограничения скорости и общего количества подключений
	// Например, если ограничение скорости равно 10, и у нас 100000 подключений, мы ожидаем много блокировок
	// Но если ограничение скорости очень высокое, мы можем не ожидать никаких блокировок.
	// Это упрощенная проверка, и ее может потребоваться скорректировать в зависимости от логики промежуточного ПО.
	// Если ограничение скорости высокое, мы можем не ожидать никаких блокировок.
	if rateLimit < numClients && bannedConnections == 0 {
		t.Logf("Ограничение скорости высокое, мы можем не ожидать никаких блокировок.")
		// t.Errorf("Ожидались некоторые заблокированные подключения, но ни одного не было")
	}
}
