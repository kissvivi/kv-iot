package v1

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/publicdevice"
	"kv-iot/datacenter/runtime"
	"kv-iot/datacenter/state"
	"kv-iot/datacenter/storage"
)

type ChannelController interface {
	List() []runtime.ChannelSnapshot
	Snapshot(channelID string) (runtime.ChannelSnapshot, bool)
	StartChannel(ctx context.Context, spec runtime.ChannelSpec) error
	StopChannel(ctx context.Context, channelID string) error
	RestartChannel(ctx context.Context, channelID string) error
	TestChannel(ctx context.Context, channelID string, points []protocol.PointBinding) ([]protocol.PointSample, error)
}

type CommandController interface {
	Submit(ctx context.Context, request runtime.CommandRequest) (*runtime.Command, bool, error)
	Get(ctx context.Context, requestID string) (*runtime.Command, error)
}

type EventReader interface {
	List(ctx context.Context, filter runtime.EventFilter) ([]runtime.Event, error)
}

type RuntimeSpecResolver interface {
	LoadChannelSpec(ctx context.Context, channelID string) (runtime.ChannelSpec, error)
	ResolveCommandPoint(
		ctx context.Context,
		deviceID int64,
		point protocol.PointBinding,
	) (string, protocol.PointBinding, error)
	ResolveDevicePointBinding(
		ctx context.Context,
		deviceID int64,
		bindingID int64,
	) (string, protocol.PointBinding, error)
	ListChannelIDsByProfile(ctx context.Context, profileID int64) ([]string, error)
}

type API struct {
	channels  ChannelController
	state     state.StateStore
	commands  CommandController
	events    EventReader
	resolver  RuntimeSpecResolver
	telemetry storage.SeriesReader
	mobile    publicdevice.Catalog
}

func NewAPI(channels ChannelController, stateStore state.StateStore, commands CommandController, events EventReader) (*API, error) {
	if channels == nil {
		return nil, fmt.Errorf("channel controller is required")
	}
	if stateStore == nil {
		return nil, fmt.Errorf("state store is required")
	}
	if commands == nil {
		return nil, fmt.Errorf("command controller is required")
	}
	if events == nil {
		return nil, fmt.Errorf("event reader is required")
	}
	return &API{
		channels: channels,
		state:    stateStore,
		commands: commands,
		events:   events,
	}, nil
}

func (a *API) SetRuntimeSpecResolver(resolver RuntimeSpecResolver) {
	a.resolver = resolver
}

func (a *API) SetTelemetryReader(reader storage.SeriesReader) {
	a.telemetry = reader
}

func (a *API) SetPublicDeviceCatalog(catalog publicdevice.Catalog) {
	a.mobile = catalog
}

func (a *API) RegisterPublicRoutes(router gin.IRouter) {
	devices := router.Group("/runtime/public/devices")
	{
		devices.GET("/:id", a.getPublicDevice)
		devices.POST("/:id/commands", a.createPublicCommand)
		devices.GET("/:id/commands/:requestId", a.getPublicCommand)
	}
}

func (a *API) RegisterRoutes(router gin.IRouter) {
	router.GET("/ping", a.ping)

	channels := router.Group("/channels")
	{
		channels.GET("", a.listChannels)
		channels.POST("", a.startChannel)
		channels.POST("/start", a.startChannel)
		channels.GET("/:id/status", a.getChannelStatus)
		channels.POST("/:id/start", a.startChannel)
		channels.POST("/:id/stop", a.stopChannel)
		channels.POST("/:id/restart", a.restartChannel)
		channels.POST("/:id/test", a.testChannel)
	}

	devices := router.Group("/devices")
	{
		devices.GET("/:id/state", a.listDeviceState)
		devices.GET("/:id/state/:point", a.getDeviceState)
		devices.POST("/:id/commands", a.createCommand)
		devices.POST("/:id/share", a.createDeviceShare)
		devices.POST("/:id/point-bindings/:bindingId/test", a.testDevicePointBinding)
	}

	router.GET("/devices/:id/telemetry", a.listTelemetry)
	router.POST("/profiles/:id/reload", a.reloadProfileChannels)
	router.GET("/commands/:requestId", a.getCommand)
	router.GET("/events", a.listEvents)
}

func (a *API) ping(c *gin.Context) {
	writeSuccess(c, 200, gin.H{"status": "ok"}, "runtime api is ready")
}
