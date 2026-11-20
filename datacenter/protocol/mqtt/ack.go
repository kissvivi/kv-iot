package mqtt

import (
	"fmt"
	"strings"
)

type ackWaiter struct {
	topicFilter      string
	correlationField string
	requestID        string
	result           chan error
}

func (d *Driver) registerAckWaiter(requestID, topicFilter, correlationField string) <-chan error {
	waiter := &ackWaiter{
		topicFilter:      topicFilter,
		correlationField: correlationField,
		requestID:        requestID,
		result:           make(chan error, 1),
	}
	d.mu.Lock()
	d.waiters[requestID] = waiter
	d.mu.Unlock()
	return waiter.result
}

func (d *Driver) removeAckWaiter(requestID string) {
	d.mu.Lock()
	delete(d.waiters, requestID)
	d.mu.Unlock()
}

func (d *Driver) resolveAck(message Message) {
	d.mu.RLock()
	waiters := make([]*ackWaiter, 0, len(d.waiters))
	for _, waiter := range d.waiters {
		waiters = append(waiters, waiter)
	}
	d.mu.RUnlock()
	if len(waiters) == 0 {
		return
	}

	for _, waiter := range waiters {
		if !matchTopicFilter(waiter.topicFilter, message.Topic) {
			continue
		}
		if !ackMatches(waiter, message.Payload) {
			continue
		}
		err := ackError(message.Payload)
		select {
		case waiter.result <- err:
		default:
		}
		d.removeAckWaiter(waiter.requestID)
	}
}

func ackMatches(waiter *ackWaiter, payload []byte) bool {
	document, err := decodeJSONValue(payload)
	if err != nil {
		return strings.TrimSpace(string(payload)) == waiter.requestID
	}
	value, exists := getPath(document, waiter.correlationField)
	if !exists {
		return false
	}
	return valueToString(value) == waiter.requestID
}

func ackError(payload []byte) error {
	document, err := decodeJSONValue(payload)
	if err != nil {
		return nil
	}
	object, ok := document.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"ok", "success"} {
		if value, exists := object[key]; exists {
			success, ok := value.(bool)
			if ok && !success {
				message := valueToString(object["message"])
				if message == "" {
					message = valueToString(object["error"])
				}
				if message == "" {
					message = "device returned a negative acknowledgement"
				}
				return fmt.Errorf("%w: %s", ErrAckRejected, message)
			}
		}
	}
	return nil
}
