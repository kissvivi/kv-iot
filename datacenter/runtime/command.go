package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusRunning   CommandStatus = "running"
	CommandStatusSucceeded CommandStatus = "succeeded"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
	CommandStatusCanceled  CommandStatus = "canceled"
)

var (
	ErrCommandNotFound   = errors.New("command not found")
	ErrCommandInvalid    = errors.New("invalid command request")
	ErrCommandInProgress = errors.New("command is still in progress")
)

func (s CommandStatus) Terminal() bool {
	switch s {
	case CommandStatusSucceeded, CommandStatusFailed, CommandStatusTimeout, CommandStatusCanceled:
		return true
	default:
		return false
	}
}

type CommandRequest struct {
	RequestID   string                `json:"request_id"`
	ChannelID   string                `json:"channel_id"`
	DeviceID    int64                 `json:"device_id"`
	DeviceNo    string                `json:"device_no,omitempty"`
	Point       protocol.PointBinding `json:"point"`
	Value       any                   `json:"value,omitempty"`
	Timeout     time.Duration         `json:"timeout,omitempty"`
	Readback    bool                  `json:"readback,omitempty"`
	RequestedBy string                `json:"requested_by,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	Metadata    map[string]string     `json:"metadata,omitempty"`
}

type Command struct {
	RequestID    string                `json:"request_id"`
	ChannelID    string                `json:"channel_id"`
	DeviceID     int64                 `json:"device_id"`
	DeviceNo     string                `json:"device_no,omitempty"`
	Point        protocol.PointBinding `json:"point"`
	Value        any                   `json:"value,omitempty"`
	Status       CommandStatus         `json:"status"`
	Result       any                   `json:"result,omitempty"`
	ErrorMessage string                `json:"error_message,omitempty"`
	RequestedBy  string                `json:"requested_by,omitempty"`
	Reason       string                `json:"reason,omitempty"`
	Metadata     map[string]string     `json:"metadata,omitempty"`
	Timeout      time.Duration         `json:"timeout"`
	Readback     bool                  `json:"readback,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	StartedAt    time.Time             `json:"started_at,omitempty"`
	FinishedAt   time.Time             `json:"finished_at,omitempty"`
}

type CommandStore interface {
	CreateIfAbsent(ctx context.Context, command Command) (*Command, bool, error)
	Update(ctx context.Context, command Command) error
	Get(ctx context.Context, requestID string) (*Command, error)
	List(ctx context.Context, limit int) ([]Command, error)
}

type CommandExecutor interface {
	Write(ctx context.Context, channelID string, point protocol.PointBinding, value any) error
}

type CommandReadbackExecutor interface {
	ReadNow(ctx context.Context, channelID string, points []protocol.PointBinding) ([]protocol.PointSample, error)
}

var _ CommandExecutor = (*Manager)(nil)
var _ CommandReadbackExecutor = (*Manager)(nil)

type CommandServiceOptions struct {
	DefaultTimeout time.Duration
	Audit          AuditSink
}

type CommandService struct {
	executor       CommandExecutor
	store          CommandStore
	audit          AuditSink
	defaultTimeout time.Duration
	baseCtx        context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewCommandService(executor CommandExecutor, store CommandStore, options CommandServiceOptions) (*CommandService, error) {
	if executor == nil {
		return nil, fmt.Errorf("%w: executor is required", ErrCommandInvalid)
	}
	if store == nil {
		return nil, fmt.Errorf("%w: command store is required", ErrCommandInvalid)
	}
	if options.DefaultTimeout <= 0 {
		options.DefaultTimeout = 5 * time.Second
	}
	if options.Audit == nil {
		options.Audit = NoopAuditSink{}
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	return &CommandService{
		executor:       executor,
		store:          store,
		audit:          options.Audit,
		defaultTimeout: options.DefaultTimeout,
		baseCtx:        baseCtx,
		cancel:         cancel,
	}, nil
}

// Submit creates a command and starts asynchronous execution. Duplicate
// request IDs return the stored command without executing again.
func (s *CommandService) Submit(ctx context.Context, request CommandRequest) (*Command, bool, error) {
	if err := validateCommandRequest(request); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = GenerateRequestID()
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}

	command := Command{
		RequestID:   request.RequestID,
		ChannelID:   strings.TrimSpace(request.ChannelID),
		DeviceID:    request.DeviceID,
		DeviceNo:    request.DeviceNo,
		Point:       request.Point,
		Value:       request.Value,
		Status:      CommandStatusPending,
		RequestedBy: request.RequestedBy,
		Reason:      request.Reason,
		Metadata:    cloneStringMap(request.Metadata),
		Timeout:     timeout,
		Readback:    request.Readback,
		CreatedAt:   time.Now(),
	}
	stored, created, err := s.store.CreateIfAbsent(ctx, command)
	if err != nil {
		return nil, false, err
	}
	if !created {
		return stored, false, nil
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.execute(stored)
	}()
	return stored, true, nil
}

func (s *CommandService) Get(ctx context.Context, requestID string) (*Command, error) {
	return s.store.Get(ctx, strings.TrimSpace(requestID))
}

func (s *CommandService) Wait(ctx context.Context, requestID string) (*Command, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		command, err := s.Get(ctx, requestID)
		if err != nil {
			return nil, err
		}
		if command.Status.Terminal() {
			return command, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *CommandService) Close(ctx context.Context) error {
	s.cancel()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	if ctx == nil {
		<-done
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *CommandService) execute(command *Command) {
	startedAt := time.Now()
	running := *command
	running.Status = CommandStatusRunning
	running.StartedAt = startedAt
	_ = s.store.Update(s.baseCtx, running)
	s.audit.Record(s.baseCtx, AuditEvent{
		UserID:       command.RequestedBy,
		Action:       "command.started",
		ResourceType: "command",
		ResourceID:   command.RequestID,
		Detail:       commandAuditDetail(running),
		CreatedAt:    startedAt,
	})

	ctx, cancel := context.WithTimeout(s.baseCtx, running.Timeout)
	err := s.executor.Write(ctx, running.ChannelID, running.Point, running.Value)
	finishedAt := time.Now()
	cancel()

	completed := running
	completed.FinishedAt = finishedAt
	switch {
	case err == nil:
		completed.Status = CommandStatusSucceeded
		completed.Result = map[string]any{
			"accepted":         true,
			"channel_id":       running.ChannelID,
			"point_identifier": running.Point.PointIdentifier,
		}
		if running.Readback || running.MetaReadback() {
			completed.Result, err = s.readback(completed)
			if err != nil {
				completed.Status = CommandStatusFailed
				completed.ErrorMessage = err.Error()
				completed.Result = nil
			}
		}
	case errors.Is(err, context.DeadlineExceeded):
		completed.Status = CommandStatusTimeout
		completed.ErrorMessage = err.Error()
	case errors.Is(err, context.Canceled):
		completed.Status = CommandStatusCanceled
		completed.ErrorMessage = err.Error()
	default:
		completed.Status = CommandStatusFailed
		completed.ErrorMessage = err.Error()
	}

	_ = s.store.Update(context.Background(), completed)
	s.audit.Record(context.Background(), AuditEvent{
		UserID:       command.RequestedBy,
		Action:       "command.finished",
		ResourceType: "command",
		ResourceID:   command.RequestID,
		Detail:       commandAuditDetail(completed),
		CreatedAt:    finishedAt,
	})
}

func (s *CommandService) readback(command Command) (any, error) {
	reader, ok := s.executor.(CommandReadbackExecutor)
	if !ok {
		return map[string]any{"readback_skipped": true}, nil
	}
	ctx, cancel := context.WithTimeout(s.baseCtx, command.Timeout)
	defer cancel()
	return reader.ReadNow(ctx, command.ChannelID, []protocol.PointBinding{command.Point})
}

func (c Command) MetaReadback() bool {
	if c.Metadata == nil {
		return false
	}
	return strings.EqualFold(c.Metadata["readback"], "true")
}

func validateCommandRequest(request CommandRequest) error {
	if strings.TrimSpace(request.ChannelID) == "" {
		return fmt.Errorf("%w: channel_id is required", ErrCommandInvalid)
	}
	if err := request.Point.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrCommandInvalid, err)
	}
	return nil
}

func commandAuditDetail(command Command) map[string]any {
	return map[string]any{
		"request_id":       command.RequestID,
		"channel_id":       command.ChannelID,
		"device_id":        command.DeviceID,
		"point_identifier": command.Point.PointIdentifier,
		"status":           command.Status,
		"value":            command.Value,
		"reason":           command.Reason,
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}
