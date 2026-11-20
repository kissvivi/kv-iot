package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"kv-iot/datacenter/protocol"
)

const DefaultPointMeasurement = "point_value"

type InfluxWriterConfig struct {
	Org         string
	Bucket      string
	Measurement string
	OnError     func(error)
}

// InfluxWriter owns one Influx client and one WriteAPI for its lifetime.
// Sample writes therefore do not create clients or writers per point.
type InfluxWriter struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	org         string
	bucket      string
	measurement string
	onError     func(error)
	done        chan struct{}
	closeOnce   sync.Once

	mu        sync.RWMutex
	lastError error
}

var _ SeriesWriter = (*InfluxWriter)(nil)
var _ SeriesReader = (*InfluxWriter)(nil)

// NewInfluxWriter creates a reusable writer around an existing client.
func NewInfluxWriter(client influxdb2.Client, config InfluxWriterConfig) (*InfluxWriter, error) {
	if client == nil {
		return nil, fmt.Errorf("influx client is required")
	}
	if strings.TrimSpace(config.Org) == "" {
		return nil, fmt.Errorf("influx org is required")
	}
	if strings.TrimSpace(config.Bucket) == "" {
		return nil, fmt.Errorf("influx bucket is required")
	}
	writer := client.WriteAPI(config.Org, config.Bucket)
	return NewInfluxWriterWithAPI(writer, config)
}

// NewInfluxWriterWithAPI injects an existing WriteAPI. It is useful for tests
// and for applications that already own a client lifecycle.
func NewInfluxWriterWithAPI(writeAPI api.WriteAPI, config InfluxWriterConfig) (*InfluxWriter, error) {
	if writeAPI == nil {
		return nil, fmt.Errorf("influx write API is required")
	}
	measurement := strings.TrimSpace(config.Measurement)
	if measurement == "" {
		measurement = DefaultPointMeasurement
	}
	if config.OnError == nil {
		config.OnError = func(error) {}
	}
	w := &InfluxWriter{
		writeAPI:    writeAPI,
		org:         strings.TrimSpace(config.Org),
		bucket:      strings.TrimSpace(config.Bucket),
		measurement: measurement,
		onError:     config.OnError,
		done:        make(chan struct{}),
	}
	go w.collectErrors()
	return w, nil
}

// NewInfluxWriterFromConnection is a convenience constructor that creates one
// client for callers that do not already own a client lifecycle.
func NewInfluxWriterFromConnection(serverURL, token string, config InfluxWriterConfig) (*InfluxWriter, error) {
	if strings.TrimSpace(serverURL) == "" {
		return nil, fmt.Errorf("influx server URL is required")
	}
	client := influxdb2.NewClient(serverURL, token)
	writer, err := NewInfluxWriter(client, config)
	if err != nil {
		client.Close()
		return nil, err
	}
	writer.client = client
	return writer, nil
}

func (w *InfluxWriter) WritePointSamples(ctx context.Context, samples []protocol.PointSample) error {
	if w == nil || w.writeAPI == nil {
		return fmt.Errorf("influx writer is not configured")
	}
	if len(samples) == 0 {
		return nil
	}
	for _, sample := range samples {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		w.writeAPI.WritePoint(w.newPoint(sample))
	}
	w.writeAPI.Flush()
	return nil
}

// Publish makes InfluxWriter usable as a runtime.SampleSink without importing
// the runtime package.
func (w *InfluxWriter) Publish(ctx context.Context, samples []protocol.PointSample) error {
	return w.WritePointSamples(ctx, samples)
}

func (w *InfluxWriter) QueryPointSamples(ctx context.Context, request TelemetryQuery) ([]protocol.PointSample, error) {
	if w == nil || w.client == nil {
		return nil, fmt.Errorf("influx query client is not configured")
	}
	if strings.TrimSpace(w.org) == "" || strings.TrimSpace(w.bucket) == "" {
		return nil, fmt.Errorf("influx org and bucket are required")
	}
	limit := request.Limit
	if limit <= 0 || limit > 5000 {
		limit = 2000
	}
	rangeStart := "-1h"
	if !request.From.IsZero() {
		rangeStart = fmt.Sprintf("time(v: %q)", request.From.UTC().Format(time.RFC3339Nano))
	}
	rangeStop := "now()"
	if !request.To.IsZero() {
		rangeStop = fmt.Sprintf("time(v: %q)", request.To.UTC().Format(time.RFC3339Nano))
	}
	conditions := []string{
		fmt.Sprintf("r._measurement == %q", w.measurement),
		`(r._field == "value_num" or r._field == "value_bool" or r._field == "value_string")`,
	}
	if request.DeviceID > 0 {
		conditions = append(conditions, fmt.Sprintf("r.device_id == %q", strconv.FormatInt(request.DeviceID, 10)))
	}
	if identifier := strings.TrimSpace(request.PointIdentifier); identifier != "" {
		conditions = append(conditions, fmt.Sprintf("r.point_identifier == %q", identifier))
	}
	flux := fmt.Sprintf(`from(bucket: %q)
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => %s)
  |> sort(columns: ["_time"])
  |> limit(n: %d)`,
		w.bucket,
		rangeStart,
		rangeStop,
		strings.Join(conditions, " and "),
		limit,
	)
	result, err := w.client.QueryAPI(w.org).Query(ctx, flux)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	samples := make([]protocol.PointSample, 0)
	for result.Next() {
		record := result.Record()
		if record == nil {
			continue
		}
		values := record.Values()
		sample := protocol.PointSample{
			ChannelID:       stringValue(values["channel_id"]),
			ProductID:       int64Value(values["product_id"]),
			ProductKey:      stringValue(values["product_key"]),
			DeviceID:        int64Value(values["device_id"]),
			DeviceNo:        stringValue(values["device_no"]),
			PointID:         int64Value(values["point_id"]),
			PointIdentifier: stringValue(values["point_identifier"]),
			PointKind:       stringValue(values["point_kind"]),
			DataType:        stringValue(values["data_type"]),
			Value:           record.Value(),
			Quality:         stringValue(values["quality"]),
			Timestamp:       record.Time(),
		}
		samples = append(samples, sample)
	}
	if err := result.Err(); err != nil {
		return nil, err
	}
	return samples, nil
}

func (w *InfluxWriter) LastError() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastError
}

func (w *InfluxWriter) Ping(ctx context.Context) error {
	if w == nil || w.client == nil {
		return fmt.Errorf("influx client is not configured")
	}
	_, err := w.client.Ping(ctx)
	return err
}

func (w *InfluxWriter) Close() error {
	w.closeOnce.Do(func() {
		close(w.done)
		if w.client != nil {
			w.client.Close()
		}
	})
	return nil
}

func (w *InfluxWriter) collectErrors() {
	errorsCh := w.writeAPI.Errors()
	for {
		select {
		case err, open := <-errorsCh:
			if !open {
				return
			}
			if err == nil {
				continue
			}
			w.mu.Lock()
			w.lastError = err
			w.mu.Unlock()
			w.onError(err)
		case <-w.done:
			return
		}
	}
}

func (w *InfluxWriter) newPoint(sample protocol.PointSample) *write.Point {
	point := influxdb2.NewPointWithMeasurement(w.measurement)
	productKey := sample.ProductKey
	if productKey == "" {
		productKey = "unknown"
	}
	point.AddTag("product_key", productKey)
	point.AddTag("device_no", sample.DeviceNo)
	point.AddTag("point_identifier", sample.PointIdentifier)
	if sample.ProductID > 0 {
		point.AddTag("product_id", strconv.FormatInt(sample.ProductID, 10))
	}
	if sample.DeviceID > 0 {
		point.AddTag("device_id", strconv.FormatInt(sample.DeviceID, 10))
	}
	if sample.PointID > 0 {
		point.AddTag("point_id", strconv.FormatInt(sample.PointID, 10))
	}
	addTag(point, "point_kind", sample.PointKind)
	addTag(point, "data_type", sample.DataType)
	addTag(point, "channel_id", sample.ChannelID)
	addTag(point, "quality", sample.Quality)

	addValueField(point, sample.Value)
	timestamp := sample.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	point.SetTime(timestamp)
	return point
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}

func addTag(point *write.Point, key, value string) {
	if strings.TrimSpace(value) != "" {
		point.AddTag(key, value)
	}
}

func addValueField(point *write.Point, value any) {
	switch typed := value.(type) {
	case nil:
		point.AddField("value_null", "null")
	case bool:
		point.AddField("value_bool", typed)
	case string:
		point.AddField("value_string", typed)
	case int:
		point.AddField("value_num", int64(typed))
	case int8:
		point.AddField("value_num", int64(typed))
	case int16:
		point.AddField("value_num", int64(typed))
	case int32:
		point.AddField("value_num", int64(typed))
	case int64:
		point.AddField("value_num", typed)
	case uint:
		point.AddField("value_num", uint64(typed))
	case uint8:
		point.AddField("value_num", uint64(typed))
	case uint16:
		point.AddField("value_num", uint64(typed))
	case uint32:
		point.AddField("value_num", uint64(typed))
	case uint64:
		point.AddField("value_num", typed)
	case float32:
		point.AddField("value_num", float64(typed))
	case float64:
		point.AddField("value_num", typed)
	case json.Number:
		if number, err := typed.Float64(); err == nil {
			point.AddField("value_num", number)
		} else {
			point.AddField("value_string", typed.String())
		}
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			point.AddField("value_string", fmt.Sprintf("%v", value))
			return
		}
		point.AddField("value_string", string(encoded))
	}
}
