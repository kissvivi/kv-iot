package tcpframe

import "errors"

var (
	ErrNotConnected       = errors.New("tcp frame: not connected")
	ErrClosed             = errors.New("tcp frame: driver is closed")
	ErrTimeout            = errors.New("tcp frame: timeout")
	ErrConnectionClosed   = errors.New("tcp frame: connection closed")
	ErrConnectionUpstream = errors.New("tcp frame: connection error")
	ErrInvalidBinding     = errors.New("tcp frame: invalid point binding")
	ErrInvalidFrame       = errors.New("tcp frame: invalid frame")
	ErrFrameTooLarge      = errors.New("tcp frame: frame exceeds limit")
	ErrChecksum           = errors.New("tcp frame: checksum mismatch")
	ErrResponseMismatch   = errors.New("tcp frame: response does not match binding")
)
