package state

import (
	"context"
	"fmt"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

// GoRedisExecutor adapts go-redis to the small RedisExecutor contract used by
// RedisStore. This keeps state serialization independent of the client API.
type GoRedisExecutor struct {
	client *redis.Client
}

func NewGoRedisExecutor(addr, username, password string, db int) (*GoRedisExecutor, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	var options *redis.Options
	var err error
	if strings.Contains(addr, "://") {
		options, err = redis.ParseURL(addr)
		if err != nil {
			return nil, fmt.Errorf("parse redis url: %w", err)
		}
		options.Username = username
		if password != "" {
			options.Password = password
		}
		options.DB = db
	} else {
		options = &redis.Options{
			Addr:     addr,
			Username: username,
			Password: password,
			DB:       db,
		}
	}

	return &GoRedisExecutor{client: redis.NewClient(options)}, nil
}

func (e *GoRedisExecutor) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("redis executor is not configured")
	}
	value, err := e.client.Do(ctx, args...).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return value, err
}

func (e *GoRedisExecutor) Ping(ctx context.Context) error {
	if e == nil || e.client == nil {
		return fmt.Errorf("redis executor is not configured")
	}
	return e.client.Ping(ctx).Err()
}

func (e *GoRedisExecutor) Close() error {
	if e == nil || e.client == nil {
		return nil
	}
	return e.client.Close()
}
