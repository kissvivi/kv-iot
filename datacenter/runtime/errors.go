package runtime

import "errors"

var (
	ErrInvalidChannelSpec     = errors.New("invalid channel specification")
	ErrChannelAlreadyRunning  = errors.New("channel worker is already running")
	ErrEndpointAlreadyRunning = errors.New("channel endpoint already has a worker")
	ErrChannelNotFound        = errors.New("channel worker not found")
	ErrWorkerNotRunning       = errors.New("channel worker is not running")
)
