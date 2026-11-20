package runtime

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryCommandStore struct {
	mu       sync.RWMutex
	commands map[string]Command
}

func NewMemoryCommandStore() *MemoryCommandStore {
	return &MemoryCommandStore{commands: make(map[string]Command)}
}

func (s *MemoryCommandStore) CreateIfAbsent(ctx context.Context, command Command) (*Command, bool, error) {
	if err := contextError(ctx); err != nil {
		return nil, false, err
	}
	requestID := strings.TrimSpace(command.RequestID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, exists := s.commands[requestID]; exists {
		copy := existing
		return &copy, false, nil
	}
	copy := command
	s.commands[requestID] = copy
	return &copy, true, nil
}

func (s *MemoryCommandStore) Update(ctx context.Context, command Command) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	requestID := strings.TrimSpace(command.RequestID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.commands[requestID]; !exists {
		return ErrCommandNotFound
	}
	s.commands[requestID] = command
	return nil
}

func (s *MemoryCommandStore) Get(ctx context.Context, requestID string) (*Command, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	command, exists := s.commands[strings.TrimSpace(requestID)]
	s.mu.RUnlock()
	if !exists {
		return nil, ErrCommandNotFound
	}
	copy := command
	return &copy, nil
}

func (s *MemoryCommandStore) List(ctx context.Context, limit int) ([]Command, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	commands := make([]Command, 0, len(s.commands))
	for _, command := range s.commands {
		commands = append(commands, command)
	}
	s.mu.RUnlock()
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].CreatedAt.After(commands[j].CreatedAt)
	})
	if limit > 0 && len(commands) > limit {
		commands = commands[:limit]
	}
	return commands, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
