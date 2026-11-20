package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	authservice "kv-iot/auth/service"
	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/publicdevice"
	"kv-iot/datacenter/runtime"
	"kv-iot/datacenter/state"
	"kv-iot/datacenter/storage"
)

type fakeChannels struct {
	mu        sync.Mutex
	snapshots map[string]runtime.ChannelSnapshot
	started   int
	stopped   int
	restarted int
}

func newFakeChannels() *fakeChannels {
	return &fakeChannels{snapshots: make(map[string]runtime.ChannelSnapshot)}
}

func (f *fakeChannels) List() []runtime.ChannelSnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]runtime.ChannelSnapshot, 0, len(f.snapshots))
	for _, snapshot := range f.snapshots {
		result = append(result, snapshot)
	}
	return result
}

func (f *fakeChannels) Snapshot(channelID string) (runtime.ChannelSnapshot, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	snapshot, exists := f.snapshots[channelID]
	return snapshot, exists
}

func (f *fakeChannels) StartChannel(ctx context.Context, spec runtime.ChannelSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started++
	f.snapshots[spec.ChannelID] = runtime.ChannelSnapshot{
		ChannelID:    spec.ChannelID,
		ProtocolType: spec.ProtocolType,
		Status:       runtime.ChannelStatusRunning,
	}
	return nil
}

func (f *fakeChannels) StopChannel(ctx context.Context, channelID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.snapshots[channelID]; !exists {
		return runtime.ErrChannelNotFound
	}
	f.stopped++
	delete(f.snapshots, channelID)
	return nil
}

func (f *fakeChannels) RestartChannel(ctx context.Context, channelID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.snapshots[channelID]; !exists {
		return runtime.ErrChannelNotFound
	}
	f.restarted++
	snapshot := f.snapshots[channelID]
	snapshot.Status = runtime.ChannelStatusRunning
	f.snapshots[channelID] = snapshot
	return nil
}

func (f *fakeChannels) TestChannel(ctx context.Context, channelID string, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	return []protocol.PointSample{{ChannelID: channelID, PointIdentifier: "test", Quality: protocol.QualityGood}}, nil
}

type apiTestExecutor struct{}

func (apiTestExecutor) Write(ctx context.Context, channelID string, point protocol.PointBinding, value any) error {
	return nil
}

type fakeMobileCatalog struct {
	device publicdevice.Device
}

func (f fakeMobileCatalog) LoadMobileDevice(ctx context.Context, deviceID int64) (publicdevice.Device, error) {
	if int64(f.device.ID) != deviceID {
		return publicdevice.Device{}, publicdevice.ErrNotFound
	}
	return f.device, nil
}

type fakeRuntimeResolver struct{}

func (fakeRuntimeResolver) LoadChannelSpec(ctx context.Context, channelID string) (runtime.ChannelSpec, error) {
	return runtime.ChannelSpec{}, nil
}

func (fakeRuntimeResolver) ResolveCommandPoint(
	ctx context.Context,
	deviceID int64,
	point protocol.PointBinding,
) (string, protocol.PointBinding, error) {
	point.DeviceID = deviceID
	return "channel-1", point, nil
}

func (fakeRuntimeResolver) ResolveDevicePointBinding(
	ctx context.Context,
	deviceID int64,
	bindingID int64,
) (string, protocol.PointBinding, error) {
	return "channel-1", protocol.PointBinding{
		DeviceID:        deviceID,
		PointIdentifier: "binding",
	}, nil
}

func (fakeRuntimeResolver) ListChannelIDsByProfile(ctx context.Context, profileID int64) ([]string, error) {
	return nil, nil
}

func newTestRouter(t *testing.T) (*gin.Engine, *runtime.CommandService, state.StateStore) {
	t.Helper()
	channels := newFakeChannels()
	stateStore := state.NewMemoryStore()
	commandService, err := runtime.NewCommandService(apiTestExecutor{}, runtime.NewMemoryCommandStore(), runtime.CommandServiceOptions{})
	if err != nil {
		t.Fatalf("new command service: %v", err)
	}
	events := runtime.NewMemoryEventStore()
	api, err := NewAPI(channels, stateStore, commandService, events)
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	return NewRouterWithMiddlewares(api), commandService, stateStore
}

func TestRuntimeChannelRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, commandService, _ := newTestRouter(t)
	defer commandService.Close(context.Background())

	body := []byte(`{"channel_id":"channel-1","protocol_type":"modbus_tcp","config":{"host":"127.0.0.1"}}`)
	response := performJSON(router, http.MethodPost, "/runtime/channels/start", body)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected start status 201, got %d: %s", response.Code, response.Body.String())
	}
	response = performJSON(router, http.MethodGet, "/runtime/channels/channel-1/status", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	response = performJSON(router, http.MethodPost, "/runtime/channels/channel-1/test", []byte(`{}`))
	if response.Code != http.StatusOK {
		t.Fatalf("expected test 200, got %d", response.Code)
	}
	response = performJSON(router, http.MethodPost, "/runtime/channels/channel-1/restart", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected restart 200, got %d", response.Code)
	}
	response = performJSON(router, http.MethodPost, "/runtime/channels/channel-1/stop", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected stop 200, got %d", response.Code)
	}
}

func TestRuntimeStateAndCommandRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, commandService, stateStore := newTestRouter(t)
	defer commandService.Close(context.Background())

	if err := stateStore.Set(context.Background(), protocol.PointSample{
		DeviceID:        1,
		PointIdentifier: "temperature",
		Value:           22.5,
		Quality:         protocol.QualityGood,
		Timestamp:       time.Now(),
	}, time.Minute); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	response := performJSON(router, http.MethodGet, "/runtime/devices/1/state/temperature", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected state 200, got %d: %s", response.Code, response.Body.String())
	}

	body := []byte(`{
		"requestId":"request-api-1",
		"channelId":"channel-1",
		"point":{"point_identifier":"switch"},
		"value":true
	}`)
	response = performJSON(router, http.MethodPost, "/runtime/devices/1/commands", body)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected command 202, got %d: %s", response.Code, response.Body.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := commandService.Wait(ctx, "request-api-1"); err != nil {
		t.Fatalf("wait command: %v", err)
	}
	response = performJSON(router, http.MethodGet, "/runtime/commands/request-api-1", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected command get 200, got %d", response.Code)
	}
}

func TestPublicDeviceShareRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, commandService, stateStore := newTestRouter(t)
	defer commandService.Close(context.Background())

	if err := stateStore.Set(context.Background(), protocol.PointSample{
		DeviceID:        7,
		PointIdentifier: "temperature",
		Value:           23.5,
		Quality:         protocol.QualityGood,
		Timestamp:       time.Now(),
	}, time.Minute); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	apiCatalog := fakeMobileCatalog{device: publicdevice.Device{
		ID:          7,
		Name:        "Hall sensor",
		DeviceNo:    "sensor-7",
		ProductID:   3,
		ProductKey:  "sensor",
		ProductName: "Temperature sensor",
		Status:      1,
		Properties: []publicdevice.Property{{
			Identifier: "temperature",
			Name:       "Temperature",
			DataType:   "float64",
			Unit:       "C",
		}},
		Actions: []publicdevice.Action{{
			ID:          11,
			Identifier:  "calibrate",
			Name:        "Calibrate",
			InputSchema: `{"type":"object"}`,
			Available:   true,
		}},
	}}

	// Registering the catalog requires a second router because the helper
	// above returns a complete router.
	api, err := NewAPI(newFakeChannels(), stateStore, commandService, runtime.NewMemoryEventStore())
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	api.SetPublicDeviceCatalog(apiCatalog)
	api.SetRuntimeSpecResolver(fakeRuntimeResolver{})
	router = NewRouterWithMiddlewares(api)

	noToken := performJSON(router, http.MethodGet, "/runtime/public/devices/7", nil)
	if noToken.Code != http.StatusUnauthorized {
		t.Fatalf("expected no-token status 401, got %d: %s", noToken.Code, noToken.Body.String())
	}

	token, _, err := authservice.GenerateDeviceShareToken(7, time.Hour)
	if err != nil {
		t.Fatalf("generate share token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/runtime/public/devices/7", nil)
	request.Header.Set("X-Device-Share-Token", token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected public snapshot 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"has_value":true`)) {
		t.Fatalf("public snapshot does not contain merged state: %s", response.Body.String())
	}

	wrongDevice := httptest.NewRequest(http.MethodGet, "/runtime/public/devices/8", nil)
	wrongDevice.Header.Set("X-Device-Share-Token", token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, wrongDevice)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected wrong-device status 403, got %d: %s", response.Code, response.Body.String())
	}

	commandRequest := httptest.NewRequest(
		http.MethodPost,
		"/runtime/public/devices/7/commands",
		bytes.NewBufferString(`{"action_identifier":"calibrate","input":{"mode":"quick"}}`),
	)
	commandRequest.Header.Set("Content-Type", "application/json")
	commandRequest.Header.Set("X-Device-Share-Token", token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, commandRequest)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected public command 202, got %d: %s", response.Code, response.Body.String())
	}
	var commandEnvelope struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &commandEnvelope); err != nil {
		t.Fatalf("decode command response: %v", err)
	}
	if commandEnvelope.Data.RequestID == "" {
		t.Fatalf("public command response has no request id: %s", response.Body.String())
	}
	waitContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := commandService.Wait(waitContext, commandEnvelope.Data.RequestID); err != nil {
		t.Fatalf("wait public command: %v", err)
	}
	commandLookup := httptest.NewRequest(
		http.MethodGet,
		"/runtime/public/devices/7/commands/"+commandEnvelope.Data.RequestID,
		nil,
	)
	commandLookup.Header.Set("X-Device-Share-Token", token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, commandLookup)
	if response.Code != http.StatusOK {
		t.Fatalf("expected public command lookup 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRuntimeTelemetryRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	history := storage.NewMemoryHistoryStore(100)
	now := time.Now().UTC()
	if err := history.WritePointSamples(context.Background(), []protocol.PointSample{{
		DeviceID:        1,
		DeviceNo:        "device-1",
		PointIdentifier: "temperature",
		Value:           23.5,
		Quality:         protocol.QualityGood,
		Timestamp:       now,
	}}); err != nil {
		t.Fatalf("seed telemetry: %v", err)
	}
	commandService, err := runtime.NewCommandService(
		apiTestExecutor{},
		runtime.NewMemoryCommandStore(),
		runtime.CommandServiceOptions{},
	)
	if err != nil {
		t.Fatalf("new command service: %v", err)
	}
	defer commandService.Close(context.Background())
	api, err := NewAPI(
		newFakeChannels(),
		state.NewMemoryStore(),
		commandService,
		runtime.NewMemoryEventStore(),
	)
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	api.SetTelemetryReader(history)
	router := NewRouterWithMiddlewares(api)

	response := performJSON(
		router,
		http.MethodGet,
		"/runtime/devices/1/telemetry?point_identifier=temperature&limit=10",
		nil,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected telemetry 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"value":23.5`)) {
		t.Fatalf("telemetry response does not contain sample: %s", response.Body.String())
	}
}

func TestRuntimeEventsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	channels := newFakeChannels()
	stateStore := state.NewMemoryStore()
	commandService, _ := runtime.NewCommandService(apiTestExecutor{}, runtime.NewMemoryCommandStore(), runtime.CommandServiceOptions{})
	defer commandService.Close(context.Background())
	events := runtime.NewMemoryEventStore()
	if err := events.Append(context.Background(), runtime.Event{
		DeviceID:        1,
		EventIdentifier: "alarm",
		Severity:        "warning",
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	api, err := NewAPI(channels, stateStore, commandService, events)
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	router := NewRouterWithMiddlewares(api)

	response := performJSON(router, http.MethodGet, "/runtime/events?deviceId=1", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var got []runtime.Event
	if err := json.Unmarshal(envelope.Data, &got); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(got) != 1 || got[0].EventIdentifier != "alarm" {
		t.Fatalf("unexpected events: %#v", got)
	}
}

func performJSON(router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
