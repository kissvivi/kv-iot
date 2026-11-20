package tcpraw

import "errors"

var (
	ErrNotConnected       = errors.New("tcp raw: not connected")
	ErrClosed             = errors.New("tcp raw: driver is closed")
	ErrTimeout            = errors.New("tcp raw: timeout")
	ErrConnectionClosed   = errors.New("tcp raw: connection closed")
	ErrConnectionUpstream = errors.New("tcp raw: connection error")
	ErrInvalidBinding     = errors.New("tcp raw: invalid point binding")
	ErrInvalidResponse    = errors.New("tcp raw: invalid response")
	ErrResponseTooLarge   = errors.New("tcp raw: response exceeds limit")
)
