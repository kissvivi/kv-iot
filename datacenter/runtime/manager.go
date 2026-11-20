package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"kv-iot/datacenter/protocol"
)

// Manager owns channel workers and prevents duplicate workers for the same
// channel or physical endpoint.
type Manager struct {
	mu sync.RWMutex

	registry          *protocol.Registry
	sink              SampleSink
	workerOptions     WorkerOptions
	workers           map[string]*Worker
	endpoints         map[string]string
	startingChannels  map[string]struct{}
	startingEndpoints map[string]struct{}
}

func NewManager(registry *protocol.Registry, sink SampleSink, options WorkerOptions) (*Manager, error) {
	if registry == nil {
		return nil, fmt.Errorf("%w: registry is required", ErrInvalidChannelSpec)
	}
	if sink == nil {
		sink = NullSink{}
	}
	return &Manager{
		registry:          registry,
		sink:              sink,
		workerOptions:     options,
		workers:           make(map[string]*Worker),
		endpoints:         make(map[string]string),
		startingChannels:  make(map[string]struct{}),
		startingEndpoints: make(map[string]struct{}),
	}, nil
}

func (m *Manager) StartChannel(ctx context.Context, spec ChannelSpec) error {
	spec.ChannelID = strings.TrimSpace(spec.ChannelID)
	spec.ProtocolType = strings.TrimSpace(spec.ProtocolType)
	if err := spec.Validate(); err != nil {
		return err
	}
	channelID := strings.TrimSpace(spec.ChannelID)
	endpointKey := spec.RuntimeEndpointKey()

	m.mu.Lock()
	if _, exists := m.workers[channelID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrChannelAlreadyRunning, channelID)
	}
	if _, exists := m.startingChannels[channelID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrChannelAlreadyRunning, channelID)
	}
	if owner, exists := m.endpoints[endpointKey]; exists && owner != channelID {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s owned by %s", ErrEndpointAlreadyRunning, endpointKey, owner)
	}
	if _, exists := m.startingEndpoints[endpointKey]; exists {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrEndpointAlreadyRunning, endpointKey)
	}
	m.startingChannels[channelID] = struct{}{}
	m.startingEndpoints[endpointKey] = struct{}{}
	m.mu.Unlock()

	release := func() {
		m.mu.Lock()
		delete(m.startingChannels, channelID)
		delete(m.startingEndpoints, endpointKey)
		m.mu.Unlock()
	}

	driver, err := m.registry.Create(spec.ProtocolType, spec.DriverConfig())
	if err != nil {
		release()
		return err
	}
	worker, err := NewWorker(spec, driver, m.sink, m.workerOptions)
	if err != nil {
		release()
		_ = driver.Close()
		return err
	}
	// The worker lifetime is owned by the manager, not by the HTTP request that
	// asked to start it. Request cancellation must not tear down the channel.
	if err := worker.Start(context.Background()); err != nil {
		release()
		_ = driver.Close()
		return err
	}

	m.mu.Lock()
	m.workers[channelID] = worker
	m.endpoints[endpointKey] = channelID
	delete(m.startingChannels, channelID)
	delete(m.startingEndpoints, endpointKey)
	m.mu.Unlock()
	return nil
}

func (m *Manager) StopChannel(ctx context.Context, channelID string) error {
	channelID = strings.TrimSpace(channelID)

	m.mu.Lock()
	worker, exists := m.workers[channelID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	m.mu.Unlock()

	err := worker.Stop(ctx)

	if err == nil {
		m.mu.Lock()
		if current, exists := m.workers[channelID]; exists && current == worker {
			delete(m.workers, channelID)
			delete(m.endpoints, worker.Spec().RuntimeEndpointKey())
		}
		m.mu.Unlock()
	}
	return err
}

func (m *Manager) RestartChannel(ctx context.Context, channelID string) error {
	w, exists := m.Get(channelID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	spec := w.Spec()
	if err := m.StopChannel(ctx, channelID); err != nil {
		return err
	}
	return m.StartChannel(ctx, spec)
}

func (m *Manager) Write(ctx context.Context, channelID string, point protocol.PointBinding, value any) error {
	worker, exists := m.Get(channelID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	return worker.Write(ctx, point, value)
}

func (m *Manager) ReadNow(ctx context.Context, channelID string, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	worker, exists := m.Get(channelID)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	return worker.ReadNow(ctx, points)
}

// TestChannel performs an on-demand read. When points are omitted the
// channel's configured poll points are used.
func (m *Manager) TestChannel(ctx context.Context, channelID string, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	worker, exists := m.Get(channelID)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	if len(points) == 0 {
		points = worker.Spec().PollPoints
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("%w: no test points configured", ErrInvalidChannelSpec)
	}
	return worker.ReadNow(ctx, points)
}

func (m *Manager) UpdatePollPoints(channelID string, points []protocol.PointBinding) error {
	worker, exists := m.Get(channelID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrChannelNotFound, channelID)
	}
	return worker.UpdatePollPoints(points)
}

func (m *Manager) Get(channelID string) (*Worker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	worker, exists := m.workers[strings.TrimSpace(channelID)]
	return worker, exists
}

func (m *Manager) Snapshot(channelID string) (ChannelSnapshot, bool) {
	worker, exists := m.Get(channelID)
	if !exists {
		return ChannelSnapshot{}, false
	}
	return worker.Snapshot(), true
}

func (m *Manager) List() []ChannelSnapshot {
	m.mu.RLock()
	workers := make([]*Worker, 0, len(m.workers))
	for _, worker := range m.workers {
		workers = append(workers, worker)
	}
	m.mu.RUnlock()

	snapshots := make([]ChannelSnapshot, 0, len(workers))
	for _, worker := range workers {
		snapshots = append(snapshots, worker.Snapshot())
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ChannelID < snapshots[j].ChannelID
	})
	return snapshots
}

func (m *Manager) StartAll(ctx context.Context, specs []ChannelSpec) error {
	var firstErr error
	for _, spec := range specs {
		if !spec.Enabled {
			continue
		}
		if err := m.StartChannel(ctx, spec); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	channelIDs := make([]string, 0, len(m.workers))
	for channelID := range m.workers {
		channelIDs = append(channelIDs, channelID)
	}
	m.mu.RUnlock()
	sort.Strings(channelIDs)

	var firstErr error
	for _, channelID := range channelIDs {
		if err := m.StopChannel(ctx, channelID); err != nil && firstErr == nil &&
			!errors.Is(err, ErrChannelNotFound) {
			firstErr = err
		}
	}
	return firstErr
}
