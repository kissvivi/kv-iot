package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"kv-iot/datacenter/protocol"
)

type fakeWriteAPI struct {
	mu       sync.Mutex
	points   []*write.Point
	flushes  int
	errorsCh chan error
}

func newFakeWriteAPI() *fakeWriteAPI {
	errorsCh := make(chan error)
	close(errorsCh)
	return &fakeWriteAPI{errorsCh: errorsCh}
}

func (w *fakeWriteAPI) WriteRecord(line string) {}

func (w *fakeWriteAPI) WritePoint(point *write.Point) {
	w.mu.Lock()
	w.points = append(w.points, point)
	w.mu.Unlock()
}

func (w *fakeWriteAPI) Flush() {
	w.mu.Lock()
	w.flushes++
	w.mu.Unlock()
}

func (w *fakeWriteAPI) Errors() <-chan error {
	return w.errorsCh
}

func (w *fakeWriteAPI) SetWriteFailedCallback(cb api.WriteFailedCallback) {
	_ = cb
}

var _ api.WriteAPI = (*fakeWriteAPI)(nil)

func TestInfluxWriterReusesOneBatchAPI(t *testing.T) {
	writeAPI := newFakeWriteAPI()
	writer, err := NewInfluxWriterWithAPI(writeAPI, InfluxWriterConfig{})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	defer writer.Close()

	samples := []protocol.PointSample{
		{
			ChannelID:       "channel-1",
			ProductKey:      "product-1",
			DeviceNo:        "device-1",
			PointIdentifier: "temperature",
			Value:           25.5,
			Quality:         protocol.QualityGood,
			Timestamp:       time.Now(),
		},
		{
			ChannelID:       "channel-1",
			ProductKey:      "product-1",
			DeviceNo:        "device-1",
			PointIdentifier: "online",
			Value:           true,
			Quality:         protocol.QualityGood,
			Timestamp:       time.Now(),
		},
	}
	if err := writer.WritePointSamples(context.Background(), samples); err != nil {
		t.Fatalf("write samples: %v", err)
	}
	if err := writer.WritePointSamples(context.Background(), samples); err != nil {
		t.Fatalf("write samples again: %v", err)
	}

	writeAPI.mu.Lock()
	defer writeAPI.mu.Unlock()
	if len(writeAPI.points) != 4 {
		t.Fatalf("expected 4 points, got %d", len(writeAPI.points))
	}
	if writeAPI.flushes != 2 {
		t.Fatalf("expected 2 flushes, got %d", writeAPI.flushes)
	}
}

func TestInfluxWriterPropagatesAsyncError(t *testing.T) {
	errorsCh := make(chan error, 1)
	received := make(chan error, 1)
	writeAPI := &fakeWriteAPI{errorsCh: errorsCh}
	writer, err := NewInfluxWriterWithAPI(writeAPI, InfluxWriterConfig{
		OnError: func(err error) {
			received <- err
		},
	})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	defer writer.Close()

	expected := context.DeadlineExceeded
	errorsCh <- expected
	select {
	case err := <-received:
		if err != expected {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async error")
	}
}
