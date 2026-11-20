package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

type commandTestExecutor struct {
	mu       sync.Mutex
	writes   int
	blocking bool
	err      error
}

func (e *commandTestExecutor) Write(ctx context.Context, channelID string, point protocol.PointBinding, value any) error {
	e.mu.Lock()
	e.writes++
	blocking := e.blocking
	err := e.err
	e.mu.Unlock()
	if blocking {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (e *commandTestExecutor) writeCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.writes
}

func TestCommandServiceIsIdempotentAndAudits(t *testing.T) {
	executor := &commandTestExecutor{}
	store := NewMemoryCommandStore()
	audit := NewMemoryAuditStore()
	service, err := NewCommandService(executor, store, CommandServiceOptions{
		DefaultTimeout: time.Second,
		Audit:          audit,
	})
	if err != nil {
		t.Fatalf("new command service: %v", err)
	}
	defer service.Close(context.Background())

	request := CommandRequest{
		RequestID:   "request-1",
		ChannelID:   "channel-1",
		DeviceID:    1,
		Point:       protocol.PointBinding{PointIdentifier: "switch", DeviceID: 1},
		Value:       true,
		RequestedBy: "user-1",
	}
	first, created, err := service.Submit(context.Background(), request)
	if err != nil {
		t.Fatalf("submit first command: %v", err)
	}
	if !created || first.RequestID != request.RequestID {
		t.Fatalf("unexpected first command: created=%v command=%#v", created, first)
	}
	second, created, err := service.Submit(context.Background(), request)
	if err != nil {
		t.Fatalf("submit duplicate command: %v", err)
	}
	if created || second.RequestID != request.RequestID {
		t.Fatalf("expected duplicate command to be returned")
	}

	command, err := service.Wait(context.Background(), request.RequestID)
	if err != nil {
		t.Fatalf("wait command: %v", err)
	}
	if command.Status != CommandStatusSucceeded {
		t.Fatalf("expected succeeded, got %s: %s", command.Status, command.ErrorMessage)
	}
	if executor.writeCount() != 1 {
		t.Fatalf("expected one write, got %d", executor.writeCount())
	}
	var events []AuditEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		events, err = audit.List(context.Background(), 10)
		if err != nil {
			t.Fatalf("list audit events: %v", err)
		}
		if len(events) == 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(events) != 2 {
		t.Fatalf("expected start and finish audit events, got %d", len(events))
	}
}

func TestCommandServiceTimeout(t *testing.T) {
	executor := &commandTestExecutor{blocking: true}
	store := NewMemoryCommandStore()
	service, err := NewCommandService(executor, store, CommandServiceOptions{
		DefaultTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new command service: %v", err)
	}
	defer service.Close(context.Background())

	_, _, err = service.Submit(context.Background(), CommandRequest{
		RequestID: "request-timeout",
		ChannelID: "channel-1",
		DeviceID:  1,
		Point:     protocol.PointBinding{PointIdentifier: "switch"},
		Value:     true,
		Timeout:   20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("submit command: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	command, err := service.Wait(ctx, "request-timeout")
	if err != nil {
		t.Fatalf("wait command: %v", err)
	}
	if command.Status != CommandStatusTimeout {
		t.Fatalf("expected timeout, got %s", command.Status)
	}
}
