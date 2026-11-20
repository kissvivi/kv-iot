package state

import (
	"context"
	"testing"
	"time"
)

func TestNewGoRedisExecutorValidation(t *testing.T) {
	if _, err := NewGoRedisExecutor("", "", "", 0); err == nil {
		t.Fatal("expected empty address to fail")
	}
}

func TestGoRedisExecutorPingWithoutServer(t *testing.T) {
	executor, err := NewGoRedisExecutor("127.0.0.1:1", "", "", 0)
	if err != nil {
		t.Fatalf("NewGoRedisExecutor() error = %v", err)
	}
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := executor.Ping(ctx); err == nil {
		t.Fatal("expected ping to fail without a redis server")
	}
}
