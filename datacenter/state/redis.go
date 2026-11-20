package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

// RedisExecutor is intentionally small so this package does not depend on a
// specific Redis client. A go-redis adapter can implement Do directly. Missing
// keys should return nil, nil or ErrStateNotFound.
type RedisExecutor interface {
	Do(ctx context.Context, args ...interface{}) (interface{}, error)
}

type RedisStoreOptions struct {
	Prefix string
}

type RedisStore struct {
	executor RedisExecutor
	prefix   string
}

func NewRedisStore(executor RedisExecutor, options RedisStoreOptions) (*RedisStore, error) {
	if executor == nil {
		return nil, fmt.Errorf("redis executor is required")
	}
	prefix := strings.Trim(strings.TrimSpace(options.Prefix), ":")
	if prefix == "" {
		prefix = "iot"
	}
	return &RedisStore{
		executor: executor,
		prefix:   prefix,
	}, nil
}

func (s *RedisStore) Set(ctx context.Context, sample protocol.PointSample, ttl time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(sample.PointIdentifier) == "" {
		return fmt.Errorf("point identifier is required")
	}
	payload, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("encode point sample: %w", err)
	}

	key := s.pointKey(sample.DeviceID, sample.PointIdentifier)
	args := []interface{}{key, payload}
	if ttl > 0 {
		args = []interface{}{key, payload, "PX", durationMillis(ttl)}
	}
	if _, err := s.executor.Do(ctx, append([]interface{}{"SET"}, args...)...); err != nil {
		return fmt.Errorf("set point state: %w", err)
	}

	indexKey := s.deviceIndexKey(sample.DeviceID)
	if _, err := s.executor.Do(ctx, "SADD", indexKey, key); err != nil {
		return fmt.Errorf("add point index: %w", err)
	}
	if ttl > 0 {
		indexTTL := ttl * 3
		if indexTTL < time.Minute {
			indexTTL = time.Minute
		}
		if _, err := s.executor.Do(ctx, "PEXPIRE", indexKey, durationMillis(indexTTL)); err != nil {
			return fmt.Errorf("expire point index: %w", err)
		}
	}
	return nil
}

func (s *RedisStore) Get(ctx context.Context, deviceID int64, pointIdentifier string) (*protocol.PointSample, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	value, err := s.executor.Do(ctx, "GET", s.pointKey(deviceID, pointIdentifier))
	if err != nil {
		if errors.Is(err, ErrStateNotFound) {
			return nil, ErrStateNotFound
		}
		return nil, fmt.Errorf("get point state: %w", err)
	}
	if isNilReply(value) {
		return nil, ErrStateNotFound
	}
	payload, ok := redisString(value)
	if !ok {
		return nil, fmt.Errorf("decode point state: unexpected redis reply %T", value)
	}

	var sample protocol.PointSample
	if err := json.Unmarshal([]byte(payload), &sample); err != nil {
		return nil, fmt.Errorf("decode point state JSON: %w", err)
	}
	return &sample, nil
}

func (s *RedisStore) ListByDevice(ctx context.Context, deviceID int64) ([]protocol.PointSample, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	indexKey := s.deviceIndexKey(deviceID)
	value, err := s.executor.Do(ctx, "SMEMBERS", indexKey)
	if err != nil {
		return nil, fmt.Errorf("list point index: %w", err)
	}
	keys, err := redisStrings(value)
	if err != nil {
		return nil, fmt.Errorf("decode point index: %w", err)
	}
	sort.Strings(keys)

	samples := make([]protocol.PointSample, 0, len(keys))
	for _, key := range keys {
		pointValue, err := s.executor.Do(ctx, "GET", key)
		if err != nil {
			if errors.Is(err, ErrStateNotFound) {
				_, _ = s.executor.Do(ctx, "SREM", indexKey, key)
				continue
			}
			return nil, fmt.Errorf("get indexed point: %w", err)
		}
		if isNilReply(pointValue) {
			_, _ = s.executor.Do(ctx, "SREM", indexKey, key)
			continue
		}
		payload, ok := redisString(pointValue)
		if !ok {
			return nil, fmt.Errorf("decode indexed point: unexpected redis reply %T", pointValue)
		}
		var sample protocol.PointSample
		if err := json.Unmarshal([]byte(payload), &sample); err != nil {
			return nil, fmt.Errorf("decode indexed point JSON: %w", err)
		}
		samples = append(samples, sample)
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].PointIdentifier < samples[j].PointIdentifier
	})
	return samples, nil
}

func (s *RedisStore) Delete(ctx context.Context, deviceID int64, pointIdentifier string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key := s.pointKey(deviceID, pointIdentifier)
	if _, err := s.executor.Do(ctx, "DEL", key); err != nil {
		return fmt.Errorf("delete point state: %w", err)
	}
	if _, err := s.executor.Do(ctx, "SREM", s.deviceIndexKey(deviceID), key); err != nil {
		return fmt.Errorf("remove point index: %w", err)
	}
	return nil
}

func (s *RedisStore) SetChannelHealth(ctx context.Context, channelID string, health protocol.ChannelHealth, ttl time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("channel id is required")
	}
	health.ChannelID = channelID
	payload, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("encode channel health: %w", err)
	}

	key := s.channelHealthKey(channelID)
	args := []interface{}{"SET", key, payload}
	if ttl > 0 {
		args = append(args, "PX", durationMillis(ttl))
	}
	if _, err := s.executor.Do(ctx, args...); err != nil {
		return fmt.Errorf("set channel health: %w", err)
	}
	return nil
}

func (s *RedisStore) GetChannelHealth(ctx context.Context, channelID string) (*protocol.ChannelHealth, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	value, err := s.executor.Do(ctx, "GET", s.channelHealthKey(channelID))
	if err != nil {
		if errors.Is(err, ErrStateNotFound) {
			return nil, ErrStateNotFound
		}
		return nil, fmt.Errorf("get channel health: %w", err)
	}
	if isNilReply(value) {
		return nil, ErrStateNotFound
	}
	payload, ok := redisString(value)
	if !ok {
		return nil, fmt.Errorf("decode channel health: unexpected redis reply %T", value)
	}
	var health protocol.ChannelHealth
	if err := json.Unmarshal([]byte(payload), &health); err != nil {
		return nil, fmt.Errorf("decode channel health JSON: %w", err)
	}
	return &health, nil
}

func (s *RedisStore) pointKey(deviceID int64, pointIdentifier string) string {
	return fmt.Sprintf("%s:state:%d:%s", s.prefix, deviceID, escapeKeyPart(pointIdentifier))
}

func (s *RedisStore) deviceIndexKey(deviceID int64) string {
	return fmt.Sprintf("%s:state:index:%d", s.prefix, deviceID)
}

func (s *RedisStore) channelHealthKey(channelID string) string {
	return fmt.Sprintf("%s:channel:health:%s", s.prefix, escapeKeyPart(channelID))
}

func escapeKeyPart(value string) string {
	replacer := strings.NewReplacer("%", "%25", ":", "%3A")
	return replacer.Replace(value)
}

func durationMillis(duration time.Duration) string {
	millis := duration.Milliseconds()
	if millis < 1 {
		millis = 1
	}
	return strconv.FormatInt(millis, 10)
}

func isNilReply(value interface{}) bool {
	if value == nil {
		return true
	}
	if err, ok := value.(error); ok && errors.Is(err, ErrStateNotFound) {
		return true
	}
	return false
}

func redisString(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return "", false
	}
}

func redisStrings(value interface{}) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []string:
		return typed, nil
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := redisString(item)
			if !ok {
				return nil, fmt.Errorf("unexpected set member type %T", item)
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unexpected set reply type %T", value)
	}
}
