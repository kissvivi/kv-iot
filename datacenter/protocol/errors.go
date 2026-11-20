package protocol

import "errors"

var (
	ErrProtocolNotRegistered   = errors.New("protocol driver is not registered")
	ErrInvalidConfig           = errors.New("invalid driver configuration")
	ErrDriverNotFound          = errors.New("driver not found")
	ErrDriverAlreadyRegistered = errors.New("protocol driver is already registered")
	ErrDriverClosed            = errors.New("driver is closed")
	ErrUnsupportedCapability   = errors.New("driver capability is not supported")
)
