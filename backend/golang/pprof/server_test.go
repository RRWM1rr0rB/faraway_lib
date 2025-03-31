package pprof

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerInitialization(t *testing.T) {
	cfg := Config{
		Host:              "localhost",
		Port:              8080,
		ReadHeaderTimeout: 5 * time.Second,
	}
	server := NewServer(cfg)

	if server.address != "localhost:8080" {
		t.Errorf("Expected address to be 'localhost:8080', got '%s'", server.address)
	}

	if server.readHeaderTimeout != 5*time.Second {
		t.Errorf("Expected readHeaderTimeout to be 5s, got '%s'", server.readHeaderTimeout)
	}
}

func TestPprofHandlers(t *testing.T) {
	server := NewServer(Config{Host: "localhost", Port: 8080, ReadHeaderTimeout: 5 * time.Second})
	go func() {
		server.Run(context.Background())
	}()

	defer server.Close()

	time.Sleep(1 * time.Second)

	// List of endpoints to test
	endpoints := []string{
		pprofURL, cmdlineURL, symbolURL, traceURL,
		goroutineURL, heapURL, threadcreateURL, blockURL,
	}

	for _, endpoint := range endpoints {
		t.Log("Testing endpoint: " + endpoint)
		resp, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			t.Fatalf("Failed to NewRequest %s: %v", endpoint, err)
		}

		rr := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(rr, resp)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	}
}
