package v1

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	authservice "kv-iot/auth/service"
	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/runtime"
)

type commandCreateRequest struct {
	RequestID       string                `json:"requestId,omitempty"`
	RequestIDSnake  string                `json:"request_id,omitempty"`
	ChannelID       string                `json:"channelId,omitempty"`
	ChannelIDSnake  string                `json:"channel_id,omitempty"`
	DeviceNo        string                `json:"deviceNo,omitempty"`
	DeviceNoSnake   string                `json:"device_no,omitempty"`
	Point           protocol.PointBinding `json:"point"`
	PointKind       string                `json:"point_kind,omitempty"`
	PointIdentifier string                `json:"point_identifier,omitempty"`
	Value           any                   `json:"value,omitempty"`
	Input           any                   `json:"input,omitempty"`
	TimeoutMs       int                   `json:"timeoutMs,omitempty"`
	TimeoutMsSnake  int                   `json:"timeout_ms,omitempty"`
	Readback        bool                  `json:"readback,omitempty"`
	RequestedBy     string                `json:"requestedBy,omitempty"`
	Reason          string                `json:"reason,omitempty"`
	Metadata        map[string]string     `json:"metadata,omitempty"`
}

func (a *API) createCommand(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	var request commandCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, "invalid command request: "+err.Error())
		return
	}
	if request.RequestID == "" {
		request.RequestID = request.RequestIDSnake
	}
	if request.ChannelID == "" {
		request.ChannelID = request.ChannelIDSnake
	}
	if request.DeviceNo == "" {
		request.DeviceNo = request.DeviceNoSnake
	}
	if request.TimeoutMs == 0 {
		request.TimeoutMs = request.TimeoutMsSnake
	}
	if request.Point.PointKind == "" {
		request.Point.PointKind = request.PointKind
	}
	if request.Point.PointIdentifier == "" {
		request.Point.PointIdentifier = request.PointIdentifier
	}
	if request.RequestedBy == "" {
		if claims, ok := authservice.ClaimsFromContext(c); ok {
			request.RequestedBy = claims.Username
		}
		if request.RequestedBy == "" {
			request.RequestedBy = strings.TrimSpace(c.GetHeader("X-User-ID"))
		}
	}
	if request.Value == nil && request.Input != nil {
		switch input := request.Input.(type) {
		case map[string]any:
			request.Value = input["value"]
		default:
			request.Value = input
		}
	}
	if a.resolver != nil && (request.ChannelID == "" || len(request.Point.Config) == 0) {
		channelID, point, err := a.resolver.ResolveCommandPoint(c.Request.Context(), deviceID, request.Point)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, err.Error())
			return
		}
		request.ChannelID = channelID
		request.Point = point
	}
	timeout := time.Duration(request.TimeoutMs) * time.Millisecond
	command, created, err := a.commands.Submit(c.Request.Context(), runtime.CommandRequest{
		RequestID:   request.RequestID,
		ChannelID:   request.ChannelID,
		DeviceID:    deviceID,
		DeviceNo:    request.DeviceNo,
		Point:       request.Point,
		Value:       request.Value,
		Timeout:     timeout,
		Readback:    request.Readback,
		RequestedBy: request.RequestedBy,
		Reason:      request.Reason,
		Metadata:    request.Metadata,
	})
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	status := http.StatusOK
	message := "command already exists"
	if created {
		status = http.StatusAccepted
		message = "command accepted"
	}
	writeSuccess(c, status, command, message)
}

func (a *API) getCommand(c *gin.Context) {
	command, err := a.commands.Get(c.Request.Context(), c.Param("requestId"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, command, "command")
}
