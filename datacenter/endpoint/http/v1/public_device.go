package v1

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	authservice "kv-iot/auth/service"
	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/publicdevice"
	"kv-iot/datacenter/runtime"
	"kv-iot/device/data"
)

type publicDeviceSnapshot struct {
	DeviceID    uint                   `json:"device_id"`
	Name        string                 `json:"name"`
	DeviceNo    string                 `json:"device_no"`
	Description string                 `json:"description"`
	ProductID   int64                  `json:"product_id"`
	ProductKey  string                 `json:"product_key"`
	ProductName string                 `json:"product_name"`
	Status      int                    `json:"status"`
	LastSeenAt  *time.Time             `json:"last_seen_at,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
	Properties  []publicDeviceProperty `json:"properties"`
	Actions     []publicDeviceAction   `json:"actions"`
}

type publicDeviceProperty struct {
	Identifier   string     `json:"identifier"`
	Name         string     `json:"name"`
	DataType     string     `json:"data_type"`
	Unit         string     `json:"unit,omitempty"`
	ReadWrite    string     `json:"read_write,omitempty"`
	Precision    int        `json:"precision,omitempty"`
	MinValue     *float64   `json:"min_value,omitempty"`
	MaxValue     *float64   `json:"max_value,omitempty"`
	EnumValues   string     `json:"enum_values,omitempty"`
	DefaultValue string     `json:"default_value,omitempty"`
	Value        any        `json:"value"`
	HasValue     bool       `json:"has_value"`
	Quality      string     `json:"quality,omitempty"`
	Timestamp    *time.Time `json:"timestamp,omitempty"`
}

type publicDeviceAction struct {
	Identifier       string `json:"identifier"`
	Name             string `json:"name"`
	InputSchema      string `json:"input_schema,omitempty"`
	TimeoutMs        int    `json:"timeout_ms,omitempty"`
	ReadbackRequired bool   `json:"readback_required"`
	Available        bool   `json:"available"`
}

func (a *API) getPublicDevice(c *gin.Context) {
	deviceID, claims, ok := a.authorizePublicDevice(c)
	if !ok {
		return
	}
	if a.mobile == nil {
		writeError(c, http.StatusServiceUnavailable, codeInternal, "mobile device catalog is unavailable")
		return
	}
	device, err := a.mobile.LoadMobileDevice(c.Request.Context(), deviceID)
	if err != nil {
		writePublicDeviceError(c, err)
		return
	}
	samples, err := a.state.ListByDevice(c.Request.Context(), deviceID)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	sampleByIdentifier := make(map[string]protocol.PointSample, len(samples))
	for _, sample := range samples {
		sampleByIdentifier[sample.PointIdentifier] = sample
	}

	snapshot := publicDeviceSnapshot{
		DeviceID:    device.ID,
		Name:        device.Name,
		DeviceNo:    device.DeviceNo,
		Description: device.Description,
		ProductID:   device.ProductID,
		ProductKey:  device.ProductKey,
		ProductName: device.ProductName,
		Status:      device.Status,
		GeneratedAt: time.Now(),
		Properties:  make([]publicDeviceProperty, 0, len(device.Properties)),
		Actions:     make([]publicDeviceAction, 0, len(device.Actions)),
	}
	if !device.LastSeenAt.IsZero() {
		lastSeen := device.LastSeenAt
		snapshot.LastSeenAt = &lastSeen
	}
	for _, property := range device.Properties {
		item := publicDeviceProperty{
			Identifier:   property.Identifier,
			Name:         property.Name,
			DataType:     property.DataType,
			Unit:         property.Unit,
			ReadWrite:    property.ReadWrite,
			Precision:    property.Precision,
			MinValue:     property.MinValue,
			MaxValue:     property.MaxValue,
			EnumValues:   property.EnumValues,
			DefaultValue: property.DefaultValue,
		}
		if sample, exists := sampleByIdentifier[property.Identifier]; exists {
			item.Value = sample.Value
			item.HasValue = true
			item.Quality = string(sample.Quality)
			timestamp := sample.Timestamp
			item.Timestamp = &timestamp
		}
		snapshot.Properties = append(snapshot.Properties, item)
	}
	for _, action := range device.Actions {
		snapshot.Actions = append(snapshot.Actions, publicDeviceAction{
			Identifier:       action.Identifier,
			Name:             action.Name,
			InputSchema:      action.InputSchema,
			TimeoutMs:        action.TimeoutMs,
			ReadbackRequired: action.ReadbackRequired,
			Available:        action.Available,
		})
	}

	// Claims are validated against the URL device ID before the response is
	// written; keep the access local rather than exposing it in the payload.
	_ = claims
	writeSuccess(c, http.StatusOK, snapshot, "public device")
}

type publicCommandRequest struct {
	ActionIdentifier string `json:"action_identifier"`
	Input            any    `json:"input,omitempty"`
}

func (a *API) createPublicCommand(c *gin.Context) {
	deviceID, claims, ok := a.authorizePublicDevice(c)
	if !ok {
		return
	}
	if a.mobile == nil || a.resolver == nil {
		writeError(c, http.StatusServiceUnavailable, codeInternal, "mobile device control is unavailable")
		return
	}
	var request publicCommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, "invalid mobile command: "+err.Error())
		return
	}
	request.ActionIdentifier = strings.TrimSpace(request.ActionIdentifier)
	if request.ActionIdentifier == "" {
		writeError(c, http.StatusBadRequest, codeBadRequest, "action_identifier is required")
		return
	}

	device, err := a.mobile.LoadMobileDevice(c.Request.Context(), deviceID)
	if err != nil {
		writePublicDeviceError(c, err)
		return
	}
	action, exists := findAvailableAction(device.Actions, request.ActionIdentifier)
	if !exists {
		writeError(c, http.StatusBadRequest, codeBadRequest, "action is not available for this device")
		return
	}

	timeout := time.Duration(action.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if timeout > time.Minute {
		timeout = time.Minute
	}
	requestedBy := "device-share:" + strconv.FormatInt(claims.DeviceID, 10)
	value := publicCommandValue(request.Input)
	channelID, point, err := a.resolver.ResolveCommandPoint(
		c.Request.Context(),
		deviceID,
		protocol.PointBinding{
			PointID:         int64(action.ID),
			PointKind:       data.PointKindAction,
			PointIdentifier: action.Identifier,
		},
	)
	if err != nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, err.Error())
		return
	}
	command, created, err := a.commands.Submit(c.Request.Context(), runtime.CommandRequest{
		ChannelID:   channelID,
		DeviceID:    deviceID,
		DeviceNo:    device.DeviceNo,
		Point:       point,
		Value:       value,
		Timeout:     timeout,
		Readback:    action.ReadbackRequired,
		RequestedBy: requestedBy,
		Metadata: map[string]string{
			"source":       "device-mobile",
			"device_share": "true",
		},
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

func (a *API) getPublicCommand(c *gin.Context) {
	deviceID, claims, ok := a.authorizePublicDevice(c)
	if !ok {
		return
	}
	command, err := a.commands.Get(c.Request.Context(), c.Param("requestId"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	expectedOwner := "device-share:" + strconv.FormatInt(claims.DeviceID, 10)
	if command.DeviceID != deviceID || command.RequestedBy != expectedOwner {
		writeError(c, http.StatusNotFound, codeNotFound, "command not found")
		return
	}
	writeSuccess(c, http.StatusOK, command, "command")
}

func (a *API) authorizePublicDevice(c *gin.Context) (int64, *authservice.DeviceShareClaims, bool) {
	deviceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || deviceID <= 0 {
		writeError(c, http.StatusBadRequest, codeBadRequest, "device id must be a positive integer")
		return 0, nil, false
	}
	token := strings.TrimSpace(c.GetHeader("X-Device-Share-Token"))
	claims, err := authservice.ParseDeviceShareToken(token)
	if err != nil {
		writeError(c, http.StatusUnauthorized, codeUnauthorized, "device access link is invalid or expired")
		return 0, nil, false
	}
	if claims.DeviceID != deviceID {
		writeError(c, http.StatusForbidden, codeUnauthorized, "device access link does not match this device")
		return 0, nil, false
	}
	return deviceID, claims, true
}

func findAvailableAction(actions []publicdevice.Action, identifier string) (publicdevice.Action, bool) {
	for _, action := range actions {
		if action.Identifier == identifier && action.Available {
			return action, true
		}
	}
	return publicdevice.Action{}, false
}

func publicCommandValue(input any) any {
	values, ok := input.(map[string]any)
	if !ok {
		return input
	}
	if value, exists := values["value"]; exists {
		return value
	}
	return values
}

func writePublicDeviceError(c *gin.Context, err error) {
	if errors.Is(err, publicdevice.ErrNotFound) {
		writeError(c, http.StatusNotFound, codeNotFound, "device not found")
		return
	}
	writeError(c, http.StatusInternalServerError, codeInternal, err.Error())
}

type deviceShareRequest struct {
	TTLHours int `json:"ttl_hours"`
}

func (a *API) createDeviceShare(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	var request deviceShareRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "invalid share request: "+err.Error())
			return
		}
	}
	ttlHours := request.TTLHours
	if ttlHours <= 0 {
		ttlHours = 24
	}
	if ttlHours > 24*7 {
		writeError(c, http.StatusBadRequest, codeBadRequest, "share link may not exceed 168 hours")
		return
	}
	token, expiresAt, err := authservice.GenerateDeviceShareToken(deviceID, time.Duration(ttlHours)*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, codeInternal, "could not generate device access link")
		return
	}
	writeSuccess(c, http.StatusCreated, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"scope":      authservice.DeviceShareScope,
	}, "device access link")
}
