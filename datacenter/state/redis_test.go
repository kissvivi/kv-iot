package state

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

type fakeRedis struct {
	mu      sync.Mutex
	values  map[string]interface{}
	sets    map[string]map[string]struct{}
	expires map[string]time.Time
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{
		values:  make(map[string]interface{}),
		sets:    make(map[string]map[string]struct{}),
		expires: make(map[string]time.Time),
	}
}

func (r *fakeRedis) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("missing command")
	}
	command := strings.ToUpper(fmt.Sprint(args[0]))
	r.mu.Lock()
	defer r.mu.Unlock()

	switch command {
	case "SET":
		key := fmt.Sprint(args[1])
		value, ok := redisString(args[2])
		if !ok {
			return nil, fmt.Errorf("unexpected SET value type %T", args[2])
		}
		r.values[key] = value
		if len(args) >= 5 && strings.EqualFold(fmt.Sprint(args[3]), "PX") {
			millis, _ := time.ParseDuration(fmt.Sprint(args[4]) + "ms")
			r.expires[key] = time.Now().Add(millis)
		}
		return "OK", nil
	case "GET":
		key := fmt.Sprint(args[1])
		if expiresAt, exists := r.expires[key]; exists && !time.Now().Before(expiresAt) {
			delete(r.values, key)
			delete(r.expires, key)
		}
		return r.values[key], nil
	case "DEL":
		key := fmt.Sprint(args[1])
		delete(r.values, key)
		return 1, nil
	case "SADD":
		key := fmt.Sprint(args[1])
		if r.sets[key] == nil {
			r.sets[key] = make(map[string]struct{})
		}
		r.sets[key][fmt.Sprint(args[2])] = struct{}{}
		return 1, nil
	case "SREM":
		key := fmt.Sprint(args[1])
		delete(r.sets[key], fmt.Sprint(args[2]))
		return 1, nil
	case "SMEMBERS":
		key := fmt.Sprint(args[1])
		result := make([]interface{}, 0, len(r.sets[key]))
		for member := range r.sets[key] {
			result = append(result, member)
		}
		return result, nil
	case "PEXPIRE":
		key := fmt.Sprint(args[1])
		millis, _ := time.ParseDuration(fmt.Sprint(args[2]) + "ms")
		r.expires[key] = time.Now().Add(millis)
		return 1, nil
	default:
		return nil, fmt.Errorf("unsupported command %s", command)
	}
}

func TestRedisStoreRoundTrip(t *testing.T) {
	executor := newFakeRedis()
	store, err := NewRedisStore(executor, RedisStoreOptions{})
	if err != nil {
		t.Fatalf("new redis store: %v", err)
	}
	ctx := context.Background()
	sample := protocol.PointSample{
		ChannelID:       "channel-1",
		DeviceID:        10,
		PointIdentifier: "temperature",
		Value:           23.5,
		Quality:         protocol.QualityGood,
		Timestamp:       time.Now(),
	}
	if err := store.Set(ctx, sample, time.Minute); err != nil {
		t.Fatalf("set sample: %v", err)
	}
	got, err := store.Get(ctx, sample.DeviceID, sample.PointIdentifier)
	if err != nil {
		t.Fatalf("get sample: %v", err)
	}
	if got.PointIdentifier != sample.PointIdentifier {
		t.Fatalf("unexpected sample: %#v", got)
	}
	list, err := store.ListByDevice(ctx, sample.DeviceID)
	if err != nil {
		t.Fatalf("list samples: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one sample, got %d", len(list))
	}

	health := protocol.ChannelHealth{Status: protocol.HealthHealthy, LastCheck: time.Now()}
	if err := store.SetChannelHealth(ctx, sample.ChannelID, health, time.Minute); err != nil {
		t.Fatalf("set health: %v", err)
	}
	gotHealth, err := store.GetChannelHealth(ctx, sample.ChannelID)
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	if gotHealth.Status != protocol.HealthHealthy {
		t.Fatalf("unexpected health: %#v", gotHealth)
	}
}
