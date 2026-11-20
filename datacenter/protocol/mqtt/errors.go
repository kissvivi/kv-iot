package mqtt

import "errors"

var (
	ErrInvalidBroker     = errors.New("invalid mqtt broker")
	ErrInvalidTopic      = errors.New("invalid mqtt topic")
	ErrInvalidQoS        = errors.New("invalid mqtt qos")
	ErrNotConnected      = errors.New("mqtt client is not connected")
	ErrAckTimeout        = errors.New("mqtt ack timeout")
	ErrAckRejected       = errors.New("mqtt ack rejected")
	ErrTemplateVariable  = errors.New("mqtt template variable is missing")
	ErrUnsupportedCodec  = errors.New("unsupported mqtt payload codec")
	ErrInvalidBinding    = errors.New("invalid mqtt point binding")
	ErrSubscriberStopped = errors.New("mqtt subscriber stopped")
)
