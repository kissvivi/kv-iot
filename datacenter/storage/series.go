package storage

import (
	"context"

	"kv-iot/datacenter/protocol"
)

// SeriesWriter stores normalized samples in a time-series backend.
type SeriesWriter interface {
	WritePointSamples(ctx context.Context, samples []protocol.PointSample) error
}
