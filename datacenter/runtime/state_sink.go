package runtime

import (
	"context"
	"fmt"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/state"
)

type StateSampleSinkOptions struct {
	DefaultTTL time.Duration
	TTLFor     func(protocol.PointSample) time.Duration
}

// StateSampleSink writes normalized samples to the latest-state store.
type StateSampleSink struct {
	store      state.StateStore
	defaultTTL time.Duration
	ttlFor     func(protocol.PointSample) time.Duration
}

var _ SampleSink = (*StateSampleSink)(nil)

func NewStateSampleSink(store state.StateStore, options StateSampleSinkOptions) (*StateSampleSink, error) {
	if store == nil {
		return nil, fmt.Errorf("state store is required")
	}
	if options.DefaultTTL <= 0 {
		options.DefaultTTL = 60 * time.Second
	}
	return &StateSampleSink{
		store:      store,
		defaultTTL: options.DefaultTTL,
		ttlFor:     options.TTLFor,
	}, nil
}

func (s *StateSampleSink) Publish(ctx context.Context, samples []protocol.PointSample) error {
	var firstErr error
	for _, sample := range samples {
		ttl := s.defaultTTL
		if s.ttlFor != nil {
			if resolved := s.ttlFor(sample); resolved > 0 {
				ttl = resolved
			}
		}
		if err := s.store.Set(ctx, sample, ttl); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("write state for device %d point %s: %w",
				sample.DeviceID, sample.PointIdentifier, err)
		}
	}
	return firstErr
}

// SinkGroup fans one sample batch out to multiple sinks. Every sink is called
// even when an earlier sink fails.
type SinkGroup struct {
	Sinks []SampleSink
}

var _ SampleSink = (*SinkGroup)(nil)

func NewSinkGroup(sinks ...SampleSink) *SinkGroup {
	return &SinkGroup{Sinks: sinks}
}

func (g *SinkGroup) Publish(ctx context.Context, samples []protocol.PointSample) error {
	var firstErr error
	for _, sink := range g.Sinks {
		if sink == nil {
			continue
		}
		if err := sink.Publish(ctx, samples); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
