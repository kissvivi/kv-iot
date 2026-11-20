package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"kv-iot/datacenter/protocol"
)

// Worker owns one physical protocol connection. Reads, writes and reconnect
// attempts are serialized by opMu.
type Worker struct {
	lifecycleMu sync.Mutex
	mu          sync.RWMutex
	opMu        sync.Mutex

	spec    ChannelSpec
	driver  protocol.ProtocolDriver
	sink    SampleSink
	options WorkerOptions

	started         bool
	running         bool
	cancel          context.CancelFunc
	done            chan struct{}
	wakePoll        chan struct{}
	status          ChannelStatus
	health          protocol.ChannelHealth
	points          []protocol.PointBinding
	sequence        uint64
	startedAt       time.Time
	lastReconnectAt time.Time
	lastReadAt      time.Time
	lastSampleAt    time.Time
	lastWriteAt     time.Time
	lastErrorAt     time.Time
	lastError       string
}

func NewWorker(spec ChannelSpec, driver protocol.ProtocolDriver, sink SampleSink, options WorkerOptions) (*Worker, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if driver == nil {
		return nil, fmt.Errorf("%w: driver is required", ErrInvalidChannelSpec)
	}
	if sink == nil {
		sink = NullSink{}
	}

	now := time.Now()
	health := protocol.ChannelHealth{
		ChannelID:  spec.ChannelID,
		InstanceID: spec.InstanceID,
		Status:     protocol.HealthUnknown,
		Since:      now,
		LastCheck:  now,
	}
	return &Worker{
		spec:     cloneChannelSpec(spec),
		driver:   driver,
		sink:     sink,
		options:  options.withDefaults(),
		done:     make(chan struct{}),
		wakePoll: make(chan struct{}, 1),
		status:   ChannelStatusStopped,
		health:   health,
		points:   clonePointBindings(spec.PollPoints),
	}, nil
}

func (w *Worker) Start(parent context.Context) error {
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()

	w.mu.RLock()
	started := w.started
	running := w.running
	w.mu.RUnlock()
	if started || running {
		return ErrChannelAlreadyRunning
	}
	if parent == nil {
		parent = context.Background()
	}

	runCtx, cancel := context.WithCancel(parent)
	now := time.Now()
	w.mu.Lock()
	w.started = true
	w.status = ChannelStatusConnecting
	w.cancel = cancel
	w.done = make(chan struct{})
	w.startedAt = now
	w.lastReconnectAt = time.Time{}
	w.health.Status = protocol.HealthConnecting
	w.health.Message = "connecting"
	w.health.Since = now
	w.health.LastCheck = now
	w.mu.Unlock()

	connectCtx, connectCancel := context.WithTimeout(runCtx, w.options.ConnectTimeout)
	err := w.driver.Connect(connectCtx, w.spec.DriverConfig())
	connectCancel()
	if err != nil {
		cancel()
		_ = w.driver.Close()
		w.mu.Lock()
		w.started = false
		w.recordFailureLocked(now, err)
		w.status = ChannelStatusError
		w.mu.Unlock()
		return fmt.Errorf("connect channel %s: %w", w.spec.ChannelID, err)
	}

	w.mu.Lock()
	w.status = ChannelStatusRunning
	w.recordHealthyLocked(now)
	w.running = true
	w.mu.Unlock()

	go w.run(runCtx)
	return nil
}

func (w *Worker) run(ctx context.Context) {
	defer func() {
		close(w.done)
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	healthTicker := time.NewTicker(w.options.HealthInterval)
	defer healthTicker.Stop()

	nextRuns := make(map[string]time.Time)
	for {
		groups := w.pollGroups()
		now := time.Now()
		for name := range nextRuns {
			if _, exists := groups[name]; !exists {
				delete(nextRuns, name)
			}
		}
		for name := range groups {
			if _, exists := nextRuns[name]; !exists {
				nextRuns[name] = now
			}
		}

		var next time.Time
		for _, runAt := range nextRuns {
			if next.IsZero() || runAt.Before(next) {
				next = runAt
			}
		}
		wait := 24 * time.Hour
		if !next.IsZero() {
			wait = time.Until(next)
			if wait < 0 {
				wait = 0
			}
		}
		pollTimer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			pollTimer.Stop()
			w.mu.Lock()
			w.status = ChannelStatusStopped
			w.health.Status = protocol.HealthStopped
			w.health.Message = ctx.Err().Error()
			w.health.Since = time.Now()
			w.health.LastCheck = time.Now()
			w.mu.Unlock()
			return
		case <-w.wakePoll:
			pollTimer.Stop()
			for name := range nextRuns {
				delete(nextRuns, name)
			}
			continue
		case <-pollTimer.C:
			now = time.Now()
			for name, group := range groups {
				if runAt, exists := nextRuns[name]; exists && !now.Before(runAt) {
					w.poll(ctx, group.points)
					nextRuns[name] = now.Add(group.interval)
				}
			}
		case <-healthTicker.C:
			pollTimer.Stop()
			w.checkHealth(ctx)
		}
	}
}

func (w *Worker) poll(ctx context.Context, points []protocol.PointBinding) {
	if len(points) == 0 || !w.active() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, w.options.ReadTimeout)
	defer cancel()

	w.opMu.Lock()
	samples, err := w.driver.Read(readCtx, points)
	w.opMu.Unlock()

	now := time.Now()
	if err != nil {
		w.mu.Lock()
		w.recordFailureLocked(now, err)
		w.mu.Unlock()
		return
	}

	samples = w.normalizeSamples(samples, now)
	w.mu.Lock()
	w.status = ChannelStatusRunning
	w.recordHealthyLocked(now)
	w.lastReadAt = now
	if len(samples) > 0 {
		w.lastSampleAt = now
	}
	w.mu.Unlock()

	if len(samples) == 0 {
		return
	}
	if err := w.sink.Publish(ctx, samples); err != nil {
		w.mu.Lock()
		w.recordFailureLocked(time.Now(), fmt.Errorf("publish samples: %w", err))
		w.mu.Unlock()
	}
}

func (w *Worker) checkHealth(ctx context.Context) {
	if !w.active() {
		return
	}

	now := time.Now()
	health := w.driver.Health()
	health.ChannelID = w.spec.ChannelID
	health.InstanceID = w.spec.InstanceID
	if health.Status == "" {
		health.Status = protocol.HealthUnknown
	}
	health.LastCheck = now
	if health.Since.IsZero() {
		health.Since = now
	}

	w.mu.Lock()
	if health.LastSuccess.IsZero() {
		health.LastSuccess = w.health.LastSuccess
	}
	if health.LastError == "" {
		health.LastError = w.health.LastError
		health.LastErrorAt = w.health.LastErrorAt
	}
	w.health = health
	switch health.Status {
	case protocol.HealthHealthy:
		w.status = ChannelStatusRunning
		w.lastError = ""
	case protocol.HealthConnecting:
		w.status = ChannelStatusConnecting
	default:
		w.status = ChannelStatusDegraded
	}
	shouldReconnect := health.Status != protocol.HealthHealthy &&
		(w.lastReconnectAt.IsZero() || now.Sub(w.lastReconnectAt) >= w.options.ReconnectInterval)
	if shouldReconnect {
		w.lastReconnectAt = now
	}
	w.mu.Unlock()

	if shouldReconnect {
		w.reconnect(ctx, now)
	}
}

func (w *Worker) reconnect(ctx context.Context, now time.Time) {
	if ctx.Err() != nil {
		return
	}

	w.mu.Lock()
	w.status = ChannelStatusConnecting
	w.health.Status = protocol.HealthConnecting
	w.health.Message = "reconnecting"
	w.health.Since = now
	w.mu.Unlock()

	w.opMu.Lock()
	_ = w.driver.Close()
	connectCtx, cancel := context.WithTimeout(ctx, w.options.ConnectTimeout)
	err := w.driver.Connect(connectCtx, w.spec.DriverConfig())
	cancel()
	w.opMu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	if err != nil {
		w.recordFailureLocked(time.Now(), err)
		return
	}
	w.status = ChannelStatusRunning
	w.recordHealthyLocked(time.Now())
}

// Write serializes access to the physical connection with polling and reads.
func (w *Worker) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	if err := point.Validate(); err != nil {
		return err
	}
	if !w.active() {
		return ErrWorkerNotRunning
	}
	if ctx == nil {
		ctx = context.Background()
	}

	writeCtx, cancel := context.WithTimeout(ctx, w.options.WriteTimeout)
	defer cancel()

	w.opMu.Lock()
	err := w.driver.Write(writeCtx, point, value)
	w.opMu.Unlock()

	now := time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()
	if err != nil {
		w.recordFailureLocked(now, err)
		return err
	}
	w.lastWriteAt = now
	w.recordHealthyLocked(now)
	return nil
}

// ReadNow performs an on-demand read, primarily for command readback.
func (w *Worker) ReadNow(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	if len(points) == 0 {
		return nil, nil
	}
	for _, point := range points {
		if err := point.Validate(); err != nil {
			return nil, err
		}
	}
	if !w.active() {
		return nil, ErrWorkerNotRunning
	}
	if ctx == nil {
		ctx = context.Background()
	}

	readCtx, cancel := context.WithTimeout(ctx, w.options.ReadTimeout)
	defer cancel()

	w.opMu.Lock()
	samples, err := w.driver.Read(readCtx, points)
	w.opMu.Unlock()

	now := time.Now()
	w.mu.Lock()
	if err != nil {
		w.recordFailureLocked(now, err)
		w.mu.Unlock()
		return nil, err
	}
	w.lastReadAt = now
	w.recordHealthyLocked(now)
	w.mu.Unlock()
	return w.normalizeSamples(samples, now), nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()

	w.mu.Lock()
	running := w.running
	if !w.started && !running {
		w.status = ChannelStatusStopped
		w.health.Status = protocol.HealthStopped
		w.mu.Unlock()
		return nil
	}
	w.started = false
	w.status = ChannelStatusStopped
	w.health.Status = protocol.HealthStopped
	w.health.Message = "stopped"
	w.health.Since = time.Now()
	w.health.LastCheck = time.Now()
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if running && done != nil {
		if ctx == nil {
			<-done
		} else {
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	w.opMu.Lock()
	err := w.driver.Close()
	w.opMu.Unlock()
	if err != nil && !errors.Is(err, protocol.ErrDriverClosed) {
		return err
	}
	return nil
}

func (w *Worker) Close() error {
	return w.Stop(context.Background())
}

func (w *Worker) Done() <-chan struct{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.done
}

func (w *Worker) UpdatePollPoints(points []protocol.PointBinding) error {
	for index, point := range points {
		if err := point.Validate(); err != nil {
			return fmt.Errorf("poll point %d: %w", index, err)
		}
	}
	w.mu.Lock()
	w.points = clonePointBindings(points)
	w.mu.Unlock()
	select {
	case w.wakePoll <- struct{}{}:
	default:
	}
	return nil
}

func (w *Worker) Snapshot() ChannelSnapshot {
	w.mu.RLock()
	defer w.mu.RUnlock()
	groups := buildPollGroups(w.spec, w.points, w.options.PollInterval)
	groupSnapshots := make([]PollGroupSnapshot, 0, len(groups))
	for _, group := range groups {
		groupSnapshots = append(groupSnapshots, PollGroupSnapshot{
			Name:       group.name,
			IntervalMs: group.interval.Milliseconds(),
			PointCount: len(group.points),
		})
	}
	sort.Slice(groupSnapshots, func(i, j int) bool {
		return groupSnapshots[i].Name < groupSnapshots[j].Name
	})
	return ChannelSnapshot{
		ChannelID:    w.spec.ChannelID,
		InstanceID:   w.spec.InstanceID,
		Name:         w.spec.Name,
		ProtocolType: w.spec.ProtocolType,
		EndpointKey:  w.spec.RuntimeEndpointKey(),
		Status:       w.status,
		Health:       w.health,
		PointCount:   len(w.points),
		PollGroups:   groupSnapshots,
		StartedAt:    w.startedAt,
		LastReadAt:   w.lastReadAt,
		LastSampleAt: w.lastSampleAt,
		LastWriteAt:  w.lastWriteAt,
		LastErrorAt:  w.lastErrorAt,
		LastError:    w.lastError,
	}
}

func (w *Worker) Spec() ChannelSpec {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return cloneChannelSpec(w.spec)
}

func (w *Worker) pollPoints() []protocol.PointBinding {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return clonePointBindings(w.points)
}

type pollGroupPoints struct {
	name     string
	interval time.Duration
	points   []protocol.PointBinding
}

func (w *Worker) pollGroups() map[string]pollGroupPoints {
	w.mu.RLock()
	spec := w.spec
	points := clonePointBindings(w.points)
	w.mu.RUnlock()
	return buildPollGroups(spec, points, w.options.PollInterval)
}

func buildPollGroups(spec ChannelSpec, points []protocol.PointBinding, fallback time.Duration) map[string]pollGroupPoints {
	defaultInterval := spec.PollInterval
	if defaultInterval <= 0 {
		defaultInterval = fallback
	}
	groups := make(map[string]pollGroupPoints)
	for _, point := range points {
		name := point.PollGroup
		if name == "" {
			name = "default"
		}
		group, exists := groups[name]
		if !exists {
			interval := defaultInterval
			if configured, ok := spec.PollGroupIntervals[name]; ok && configured > 0 {
				interval = configured
			}
			group = pollGroupPoints{name: name, interval: interval}
		}
		group.points = append(group.points, point)
		groups[name] = group
	}
	return groups
}

func (w *Worker) active() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.started && (w.status == ChannelStatusRunning || w.status == ChannelStatusDegraded)
}

func (w *Worker) normalizeSamples(samples []protocol.PointSample, now time.Time) []protocol.PointSample {
	for index := range samples {
		if samples[index].ChannelID == "" {
			samples[index].ChannelID = w.spec.ChannelID
		}
		if samples[index].Quality == "" {
			samples[index].Quality = protocol.QualityGood
		}
		if samples[index].Timestamp.IsZero() {
			samples[index].Timestamp = now
		}
		if samples[index].Sequence == 0 {
			samples[index].Sequence = atomic.AddUint64(&w.sequence, 1)
		}
	}
	return samples
}

func (w *Worker) recordHealthyLocked(now time.Time) {
	w.health.Status = protocol.HealthHealthy
	w.health.Message = ""
	w.health.LastCheck = now
	w.health.LastSuccess = now
	w.health.ConsecutiveFailures = 0
	w.health.LastError = ""
}

func (w *Worker) recordFailureLocked(now time.Time, err error) {
	w.status = ChannelStatusDegraded
	w.health.Status = protocol.HealthDegraded
	w.health.Message = err.Error()
	w.health.LastCheck = now
	w.health.LastErrorAt = now
	w.health.LastError = err.Error()
	w.health.ConsecutiveFailures++
	w.lastErrorAt = now
	w.lastError = err.Error()
}

func cloneChannelSpec(spec ChannelSpec) ChannelSpec {
	spec.Config = append([]byte(nil), spec.Config...)
	spec.PollPoints = clonePointBindings(spec.PollPoints)
	if spec.PollGroupIntervals != nil {
		intervals := make(map[string]time.Duration, len(spec.PollGroupIntervals))
		for name, interval := range spec.PollGroupIntervals {
			intervals[name] = interval
		}
		spec.PollGroupIntervals = intervals
	}
	if spec.Metadata != nil {
		metadata := make(map[string]string, len(spec.Metadata))
		for key, value := range spec.Metadata {
			metadata[key] = value
		}
		spec.Metadata = metadata
	}
	return spec
}

func clonePointBindings(points []protocol.PointBinding) []protocol.PointBinding {
	if points == nil {
		return nil
	}
	cloned := make([]protocol.PointBinding, len(points))
	copy(cloned, points)
	for index := range cloned {
		cloned[index].Config = append([]byte(nil), points[index].Config...)
	}
	return cloned
}
