package modbus

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected       = errors.New("modbus connection is not established")
	ErrTimeout            = errors.New("modbus request timed out")
	ErrConnectionClosed   = errors.New("modbus connection closed")
	ErrInvalidResponse    = errors.New("invalid modbus response")
	ErrInvalidBinding     = errors.New("invalid modbus point binding")
	ErrUnsupportedArea    = errors.New("unsupported modbus area")
	ErrUnsupportedType    = errors.New("unsupported modbus data type")
	ErrReadOnlyPoint      = errors.New("modbus point is read-only")
	ErrInvalidWriteValue  = errors.New("invalid value for modbus write")
	ErrDeviceException    = errors.New("modbus device returned an exception")
	ErrIllegalFunction    = errors.New("modbus illegal function")
	ErrIllegalAddress     = errors.New("modbus illegal data address")
	ErrIllegalValue       = errors.New("modbus illegal data value")
	ErrDeviceFailure      = errors.New("modbus device failure")
	ErrAcknowledge        = errors.New("modbus request acknowledged")
	ErrDeviceBusy         = errors.New("modbus device busy")
	ErrMemoryParity       = errors.New("modbus memory parity error")
	ErrGatewayPath        = errors.New("modbus gateway path unavailable")
	ErrGatewayTarget      = errors.New("modbus gateway target failed to respond")
	ErrConnectionUpstream = errors.New("modbus transport error")
)

// Exception represents a Modbus exception response.
type Exception struct {
	FunctionCode byte
	Code         byte
}

func (e *Exception) Error() string {
	return fmt.Sprintf("%s: function=0x%02x code=0x%02x", e.Sentinel(), e.FunctionCode, e.Code)
}

func (e *Exception) Unwrap() error {
	return e.Sentinel()
}

// Sentinel returns the stable error associated with the exception code.
func (e *Exception) Sentinel() error {
	switch e.Code {
	case 0x01:
		return ErrIllegalFunction
	case 0x02:
		return ErrIllegalAddress
	case 0x03:
		return ErrIllegalValue
	case 0x04:
		return ErrDeviceFailure
	case 0x05:
		return ErrAcknowledge
	case 0x06:
		return ErrDeviceBusy
	case 0x08:
		return ErrMemoryParity
	case 0x0a:
		return ErrGatewayPath
	case 0x0b:
		return ErrGatewayTarget
	default:
		return ErrDeviceException
	}
}
