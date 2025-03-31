package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	if l == nil {
		t.Fatal("logger is nil")
	}
	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")

	ctx := context.Background()
	l = L(ctx)
	l = l.With(slog.String("test", "ok"))
	ctx = ContextWithLogger(ctx, l)
	newL := L(ctx)
	newL.Debug("debug")
	newL.Info("info")
	newL.Warn("warn")
	newL.Error("error")
}
