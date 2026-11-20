package state

import (
	"context"
	"errors"
	"time"

	"kv-iot/datacenter/protocol"
)

var ErrStateNotFound = errors.New("point state not found")

// StateStore keeps the latest point values and channel health. The channel ID
// is a string because the current device model uses a string channel ID.
type StateStore interface {
	Set(ctx context.Context, sample protocol.PointSample, ttl time.Duration) error
	Get(ctx context.Context, deviceID int64, pointIdentifier string) (*protocol.PointSample, error)
	ListByDevice(ctx context.Context, deviceID int64) ([]protocol.PointSample, error)
	Delete(ctx context.Context, deviceID int64, pointIdentifier string) error
	SetChannelHealth(ctx context.Context, channelID string, health protocol.ChannelHealth, ttl time.Duration) error
	GetChannelHealth(ctx context.Context, channelID string) (*protocol.ChannelHealth, error)
}
