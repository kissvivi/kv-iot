package configstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/runtime"
	"kv-iot/device/data"
)

// ChannelSpecResolver loads runtime channel and point bindings from the
// control-plane tables. It keeps the datacenter package independent of device
// HTTP APIs while sharing the existing MySQL schema.
type ChannelSpecResolver struct {
	db *gorm.DB
}

func NewChannelSpecResolver(db *gorm.DB) (*ChannelSpecResolver, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	return &ChannelSpecResolver{db: db}, nil
}

func (r *ChannelSpecResolver) LoadChannelSpec(ctx context.Context, channelID string) (runtime.ChannelSpec, error) {
	instanceID, err := strconv.ParseUint(strings.TrimSpace(channelID), 10, 64)
	if err != nil {
		return runtime.ChannelSpec{}, fmt.Errorf("invalid channel instance id %q: %w", channelID, err)
	}

	var instance data.ChannelInstance
	if err := r.db.WithContext(ctx).First(&instance, instanceID).Error; err != nil {
		return runtime.ChannelSpec{}, fmt.Errorf("load channel instance: %w", err)
	}

	var devices []data.Devices
	if err := r.db.WithContext(ctx).
		Where("channel_instance_id = ? AND deleted_at IS NULL", instance.ID).
		Order("id").
		Find(&devices).Error; err != nil {
		return runtime.ChannelSpec{}, fmt.Errorf("load channel devices: %w", err)
	}

	points := make([]protocol.PointBinding, 0)
	for _, device := range devices {
		devicePoints, err := r.loadDevicePoints(ctx, device, int64(instance.ID))
		if err != nil {
			return runtime.ChannelSpec{}, err
		}
		points = append(points, devicePoints...)
	}

	pollInterval, pollGroups, err := parseRuntimeTiming(instance.Config)
	if err != nil {
		return runtime.ChannelSpec{}, fmt.Errorf("parse channel timing: %w", err)
	}
	return runtime.ChannelSpec{
		ChannelID:          strconv.FormatUint(uint64(instance.ID), 10),
		InstanceID:         int64(instance.ID),
		Name:               instance.InstanceName,
		ProtocolType:       instance.ChannelType,
		DriverName:         instance.ChannelType,
		EndpointKey:        fmt.Sprintf("%s:%d", instance.ChannelType, instance.ID),
		Config:             json.RawMessage(instance.Config),
		PollInterval:       pollInterval,
		PollGroupIntervals: pollGroups,
		PollPoints:         points,
		Enabled:            instance.Enabled,
	}, nil
}

func (r *ChannelSpecResolver) ListEnabledChannelIDs(ctx context.Context) ([]string, error) {
	var instances []data.ChannelInstance
	if err := r.db.WithContext(ctx).
		Where("enabled = ? AND deleted_at IS NULL", true).
		Order("id").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("list enabled channel instances: %w", err)
	}
	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, strconv.FormatUint(uint64(instance.ID), 10))
	}
	return ids, nil
}

func (r *ChannelSpecResolver) ResolveCommandPoint(
	ctx context.Context,
	deviceID int64,
	point protocol.PointBinding,
) (string, protocol.PointBinding, error) {
	var device data.Devices
	if err := r.db.WithContext(ctx).First(&device, deviceID).Error; err != nil {
		return "", protocol.PointBinding{}, fmt.Errorf("load device: %w", err)
	}
	if device.ChannelInstanceID == 0 {
		return "", protocol.PointBinding{}, fmt.Errorf("device %d has no communication endpoint", deviceID)
	}

	resolved, err := r.resolvePointForDevice(ctx, device, device.ChannelInstanceID, point)
	if err != nil {
		return "", protocol.PointBinding{}, err
	}
	return strconv.FormatUint(uint64(device.ChannelInstanceID), 10), resolved, nil
}

func (r *ChannelSpecResolver) ResolveDevicePointBinding(
	ctx context.Context,
	deviceID int64,
	bindingID int64,
) (string, protocol.PointBinding, error) {
	var device data.Devices
	if err := r.db.WithContext(ctx).First(&device, deviceID).Error; err != nil {
		return "", protocol.PointBinding{}, fmt.Errorf("load device: %w", err)
	}
	var binding data.DevicePointBinding
	if err := r.db.WithContext(ctx).
		Where("id = ? AND device_id = ? AND deleted_at IS NULL", bindingID, deviceID).
		First(&binding).Error; err != nil {
		return "", protocol.PointBinding{}, fmt.Errorf("load device point binding: %w", err)
	}
	if !binding.Enabled {
		return "", protocol.PointBinding{}, fmt.Errorf("device point binding %d is disabled", binding.ID)
	}
	if binding.ChannelInstanceID <= 0 {
		return "", protocol.PointBinding{}, fmt.Errorf("device point binding %d has no channel", binding.ID)
	}
	var productBinding data.ProductPointBinding
	if err := r.db.WithContext(ctx).First(&productBinding, binding.ProductPointBindingID).Error; err != nil {
		return "", protocol.PointBinding{}, fmt.Errorf("load product point binding: %w", err)
	}
	if productBinding.ProfileID != device.ProfileID {
		return "", protocol.PointBinding{}, fmt.Errorf("product point binding profile does not match device profile")
	}
	point, err := r.buildPointBinding(ctx, device, productBinding, binding)
	if err != nil {
		return "", protocol.PointBinding{}, err
	}
	return strconv.FormatUint(uint64(binding.ChannelInstanceID), 10), point, nil
}

func (r *ChannelSpecResolver) ListChannelIDsByProfile(ctx context.Context, profileID int64) ([]string, error) {
	var channelIDs []int64
	if err := r.db.WithContext(ctx).
		Table("devices").
		Where("profile_id = ? AND channel_instance_id > 0 AND deleted_at IS NULL", profileID).
		Distinct().
		Pluck("channel_instance_id", &channelIDs).Error; err != nil {
		return nil, fmt.Errorf("list profile channels: %w", err)
	}
	result := make([]string, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		result = append(result, strconv.FormatInt(channelID, 10))
	}
	return result, nil
}

func (r *ChannelSpecResolver) loadDevicePoints(
	ctx context.Context,
	device data.Devices,
	channelInstanceID int64,
) ([]protocol.PointBinding, error) {
	var bindings []data.DevicePointBinding
	if err := r.db.WithContext(ctx).
		Where("device_id = ? AND channel_instance_id = ? AND enabled = ? AND deleted_at IS NULL",
			device.ID, channelInstanceID, true).
		Order("id").
		Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("load device point bindings: %w", err)
	}

	result := make([]protocol.PointBinding, 0, len(bindings))
	for _, binding := range bindings {
		point, err := r.resolveBinding(ctx, device, channelInstanceID, binding)
		if err != nil {
			if errors.Is(err, errPointNotPolled) {
				continue
			}
			return nil, err
		}
		result = append(result, point)
	}
	return result, nil
}

func (r *ChannelSpecResolver) resolvePointForDevice(
	ctx context.Context,
	device data.Devices,
	channelInstanceID int64,
	point protocol.PointBinding,
) (protocol.PointBinding, error) {
	if device.ProfileID == 0 {
		return protocol.PointBinding{}, fmt.Errorf("device %d has no protocol profile", device.ID)
	}

	var productPointBinding data.ProductPointBinding
	query := r.db.WithContext(ctx).
		Where("profile_id = ? AND point_kind = ? AND enabled = ? AND deleted_at IS NULL",
			device.ProfileID, point.PointKind, true)
	if point.PointID != 0 {
		query = query.Where("point_id = ?", point.PointID)
	} else {
		pointID, err := r.lookupPointID(ctx, device.ProductID, point.PointKind, point.PointIdentifier)
		if err != nil {
			return protocol.PointBinding{}, err
		}
		query = query.Where("point_id = ?", pointID)
	}
	if err := query.First(&productPointBinding).Error; err != nil {
		return protocol.PointBinding{}, fmt.Errorf("resolve product point binding: %w", err)
	}

	var deviceBinding data.DevicePointBinding
	if err := r.db.WithContext(ctx).
		Where("device_id = ? AND channel_instance_id = ? AND product_point_binding_id = ? AND enabled = ? AND deleted_at IS NULL",
			device.ID, channelInstanceID, productPointBinding.ID, true).
		First(&deviceBinding).Error; err != nil {
		return protocol.PointBinding{}, fmt.Errorf("resolve device point binding: %w", err)
	}

	return r.buildPointBinding(ctx, device, productPointBinding, deviceBinding)
}

func (r *ChannelSpecResolver) resolveBinding(
	ctx context.Context,
	device data.Devices,
	channelInstanceID int64,
	binding data.DevicePointBinding,
) (protocol.PointBinding, error) {
	var productPointBinding data.ProductPointBinding
	if err := r.db.WithContext(ctx).
		First(&productPointBinding, binding.ProductPointBindingID).Error; err != nil {
		return protocol.PointBinding{}, fmt.Errorf("load product point binding %d: %w", binding.ProductPointBindingID, err)
	}
	if !productPointBinding.Enabled ||
		productPointBinding.ProfileID != device.ProfileID ||
		productPointBinding.CollectionMode == data.CollectionModeOnDemand {
		return protocol.PointBinding{}, errPointNotPolled
	}
	return r.buildPointBinding(ctx, device, productPointBinding, binding)
}

func (r *ChannelSpecResolver) buildPointBinding(
	ctx context.Context,
	device data.Devices,
	productPoint data.ProductPointBinding,
	devicePoint data.DevicePointBinding,
) (protocol.PointBinding, error) {
	identifier, dataType, readOnly, err := r.lookupPointMetadata(ctx, productPoint.PointKind, int64(productPoint.PointID))
	if err != nil {
		return protocol.PointBinding{}, err
	}
	config, err := mergeJSONConfig(productPoint.BindingConfig, devicePoint.BindingOverride)
	if err != nil {
		return protocol.PointBinding{}, fmt.Errorf("merge binding config: %w", err)
	}
	return protocol.PointBinding{
		PointID:         int64(productPoint.PointID),
		ProductID:       device.ProductID,
		ProductKey:      device.ProductKey,
		DeviceID:        int64(device.ID),
		DeviceNo:        device.DeviceNo,
		PointIdentifier: identifier,
		PointKind:       productPoint.PointKind,
		CollectionMode:  productPoint.CollectionMode,
		DataType:        dataType,
		PollGroup:       productPoint.PollGroup,
		ReadOnly:        readOnly,
		Config:          config,
	}, nil
}

var errPointNotPolled = fmt.Errorf("point is configured for on-demand collection")

func parseRuntimeTiming(raw string) (time.Duration, map[string]time.Duration, error) {
	interval, err := parseDurationMillis(raw,
		"poll_interval_ms", "pollIntervalMs", "poll_interval", "pollInterval")
	if err != nil {
		return 0, nil, err
	}
	groups, err := parsePollGroupIntervals(raw)
	if err != nil {
		return 0, nil, err
	}
	return interval, groups, nil
}

func parsePollGroupIntervals(raw string) (map[string]time.Duration, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	var groupsJSON json.RawMessage
	for _, key := range []string{"poll_groups", "pollGroups", "poll_group_intervals", "pollGroupIntervals"} {
		if value, exists := values[key]; exists && len(value) > 0 && string(value) != "null" {
			groupsJSON = value
			break
		}
	}
	if len(groupsJSON) == 0 {
		return nil, nil
	}
	var groups map[string]json.RawMessage
	if err := json.Unmarshal(groupsJSON, &groups); err != nil {
		return nil, fmt.Errorf("poll groups must be an object: %w", err)
	}
	result := make(map[string]time.Duration, len(groups))
	for name, value := range groups {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("poll group name is required")
		}
		interval, err := parseDurationValue(value)
		if err != nil {
			return nil, fmt.Errorf("poll group %s: %w", name, err)
		}
		if interval <= 0 {
			return nil, fmt.Errorf("poll group %s interval must be positive", name)
		}
		result[name] = interval
	}
	return result, nil
}

func parseDurationMillis(raw string, keys ...string) (time.Duration, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return 0, err
	}
	for _, key := range keys {
		if value, exists := values[key]; exists && len(value) > 0 && string(value) != "null" {
			duration, err := parseDurationValue(value)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", key, err)
			}
			return duration, nil
		}
	}
	return 0, nil
}

func parseDurationValue(raw json.RawMessage) (time.Duration, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		value, err := time.ParseDuration(strings.TrimSpace(text))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", text, err)
		}
		return value, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0, fmt.Errorf("duration must be milliseconds or a duration string")
	}
	milliseconds, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid millisecond duration %q: %w", number.String(), err)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}

func (r *ChannelSpecResolver) lookupPointID(
	ctx context.Context,
	productID int64,
	kind string,
	identifier string,
) (uint, error) {
	if strings.TrimSpace(identifier) == "" {
		return 0, fmt.Errorf("point identifier is required")
	}
	switch kind {
	case data.PointKindProperty:
		var point data.KvProperty
		err := r.db.WithContext(ctx).
			Where("product_id = ? AND identifier = ? AND deleted_at IS NULL", productID, identifier).
			First(&point).Error
		return point.ID, err
	case data.PointKindAction:
		var point data.KvAction
		err := r.db.WithContext(ctx).
			Where("product_id = ? AND identifier = ? AND deleted_at IS NULL", productID, identifier).
			First(&point).Error
		return point.ID, err
	case data.PointKindEvent:
		var point data.KvEvent
		err := r.db.WithContext(ctx).
			Where("product_id = ? AND identifier = ? AND deleted_at IS NULL", productID, identifier).
			First(&point).Error
		return point.ID, err
	default:
		return 0, fmt.Errorf("unsupported point kind %q", kind)
	}
}

func (r *ChannelSpecResolver) lookupPointMetadata(
	ctx context.Context,
	kind string,
	pointID int64,
) (identifier string, dataType string, readOnly bool, err error) {
	switch kind {
	case data.PointKindProperty:
		var point data.KvProperty
		err = r.db.WithContext(ctx).First(&point, pointID).Error
		identifier = point.Identifier
		dataType = point.DataType
		readOnly = point.IsRw == "r"
	case data.PointKindAction:
		var point data.KvAction
		err = r.db.WithContext(ctx).First(&point, pointID).Error
		identifier = point.Identifier
		dataType = "action"
		readOnly = false
	case data.PointKindEvent:
		var point data.KvEvent
		err = r.db.WithContext(ctx).First(&point, pointID).Error
		identifier = point.Identifier
		dataType = "event"
		readOnly = true
	default:
		err = fmt.Errorf("unsupported point kind %q", kind)
	}
	if err != nil {
		return "", "", false, fmt.Errorf("load point metadata: %w", err)
	}
	return identifier, dataType, readOnly, nil
}

func mergeJSONConfig(base, override string) (json.RawMessage, error) {
	merged := make(map[string]any)
	for name, raw := range map[string]string{"base": base, "override": override} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var values map[string]any
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("%s binding config: %w", name, err)
		}
		for key, value := range values {
			merged[key] = value
		}
	}
	if len(merged) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
