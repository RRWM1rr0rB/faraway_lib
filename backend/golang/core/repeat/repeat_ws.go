package repeat

import (
	"context"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type DialFunc func(ctx context.Context) (*websocket.Conn, error)

func ConnectWithRetry(ctx context.Context, dial DialFunc, opts ...OptionSetter) (*websocket.Conn, error) {
	var conn *websocket.Conn

	operation := func(ctx context.Context, attempt int) error {
		c, err := dial(ctx)
		if err != nil {
			log.Printf("WebSocket dial failed (attempt %d): %v", attempt+1, err)
			return err
		}
		conn = c
		return nil
	}

	err := Exec(ctx, operation, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	log.Println("WebSocket connected successfully")
	return conn, nil
}
