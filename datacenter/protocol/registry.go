package protocol

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type driverRegistration struct {
	factory   DriverFactory
	validator ConfigValidator
}

// Registry stores driver factories and optional configuration validators. It
// is safe for concurrent driver creation.
type Registry struct {
	mu            sync.RWMutex
	registrations map[string]driverRegistration
}

func NewRegistry() *Registry {
	return &Registry{
		registrations: make(map[string]driverRegistration),
	}
}

// Register adds a driver with JSON-only validation. Use RegisterWithValidator
// when the protocol has a stricter configuration contract.
func (r *Registry) Register(protocolType string, factory DriverFactory) error {
	return r.RegisterWithValidator(protocolType, factory, nil)
}

// RegisterWithValidator adds a driver and its protocol-specific validator.
func (r *Registry) RegisterWithValidator(protocolType string, factory DriverFactory, validator ConfigValidator) error {
	name := normalizeProtocolType(protocolType)
	if name == "" {
		return fmt.Errorf("%w: protocol type is required", ErrInvalidConfig)
	}
	if factory == nil {
		return fmt.Errorf("%w: driver factory is required", ErrInvalidConfig)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.registrations == nil {
		r.registrations = make(map[string]driverRegistration)
	}
	if _, exists := r.registrations[name]; exists {
		return fmt.Errorf("%w: %s", ErrDriverAlreadyRegistered, name)
	}
	r.registrations[name] = driverRegistration{
		factory:   factory,
		validator: validator,
	}
	return nil
}

// MustRegister is intended for static application startup wiring.
func (r *Registry) MustRegister(protocolType string, factory DriverFactory) {
	if err := r.Register(protocolType, factory); err != nil {
		panic(err)
	}
}

// Create validates the shared config and invokes the registered factory.
func (r *Registry) Create(protocolType string, config DriverConfig) (ProtocolDriver, error) {
	name := normalizeProtocolType(protocolType)
	if name == "" {
		return nil, fmt.Errorf("%w: protocol type is required", ErrInvalidConfig)
	}
	if config.ProtocolType == "" {
		config.ProtocolType = name
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := r.Validate(name, config.Config); err != nil {
		return nil, err
	}

	r.mu.RLock()
	registration, exists := r.registrations[name]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProtocolNotRegistered, name)
	}

	driver, err := registration.factory(config)
	if err != nil {
		return nil, fmt.Errorf("create %s driver: %w", name, err)
	}
	if driver == nil {
		return nil, fmt.Errorf("%w: factory returned nil for %s", ErrDriverNotFound, name)
	}
	return driver, nil
}

// Validate performs common JSON validation and delegates to the registered
// protocol validator when present.
func (r *Registry) Validate(protocolType string, config json.RawMessage) error {
	name := normalizeProtocolType(protocolType)
	if name == "" {
		return fmt.Errorf("%w: protocol type is required", ErrInvalidConfig)
	}
	if len(strings.TrimSpace(string(config))) == 0 {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}
	if !json.Valid(config) {
		return fmt.Errorf("%w: config is not valid JSON", ErrInvalidConfig)
	}

	r.mu.RLock()
	registration, exists := r.registrations[name]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("%w: %s", ErrProtocolNotRegistered, name)
	}
	if registration.validator == nil {
		return nil
	}
	if err := registration.validator.ValidateConfig(config); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidConfig, name, err)
	}
	return nil
}

func (r *Registry) Has(protocolType string) bool {
	name := normalizeProtocolType(protocolType)
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.registrations[name]
	return exists
}

// ProtocolTypes returns registered protocol names in deterministic order.
func (r *Registry) ProtocolTypes() []string {
	r.mu.RLock()
	names := make([]string, 0, len(r.registrations))
	for name := range r.registrations {
		names = append(names, name)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	return names
}

func normalizeProtocolType(protocolType string) string {
	return strings.ToLower(strings.TrimSpace(protocolType))
}
