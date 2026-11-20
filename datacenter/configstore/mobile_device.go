package configstore

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"kv-iot/datacenter/publicdevice"
	"kv-iot/device/data"
)

// LoadMobileDevice exposes the public control model for one device. Protocol
// addresses and channel secrets are intentionally excluded.
func (r *ChannelSpecResolver) LoadMobileDevice(ctx context.Context, deviceID int64) (publicdevice.Device, error) {
	if deviceID <= 0 {
		return publicdevice.Device{}, publicdevice.ErrNotFound
	}

	var device data.Devices
	if err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", deviceID).
		First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return publicdevice.Device{}, publicdevice.ErrNotFound
		}
		return publicdevice.Device{}, fmt.Errorf("load mobile device: %w", err)
	}

	var product data.Products
	if err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", device.ProductID).
		First(&product).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return publicdevice.Device{}, fmt.Errorf("load mobile device product: %w", err)
	}

	var properties []data.KvProperty
	if err := r.db.WithContext(ctx).
		Where("product_id = ? AND deleted_at IS NULL", device.ProductID).
		Order("id").
		Find(&properties).Error; err != nil {
		return publicdevice.Device{}, fmt.Errorf("load mobile device properties: %w", err)
	}
	var actions []data.KvAction
	if err := r.db.WithContext(ctx).
		Where("product_id = ? AND deleted_at IS NULL", device.ProductID).
		Order("id").
		Find(&actions).Error; err != nil {
		return publicdevice.Device{}, fmt.Errorf("load mobile device actions: %w", err)
	}

	availableActions, err := r.availableActionIDs(ctx, device)
	if err != nil {
		return publicdevice.Device{}, err
	}

	result := publicdevice.Device{
		ID:          device.ID,
		Name:        device.Name,
		DeviceNo:    device.DeviceNo,
		Description: device.Desc,
		ProductID:   device.ProductID,
		ProductKey:  firstNonEmpty(device.ProductKey, product.ProductKey),
		ProductName: product.Name,
		Status:      device.Status,
		LastSeenAt:  device.LastSeenAt,
		Properties:  make([]publicdevice.Property, 0, len(properties)),
		Actions:     make([]publicdevice.Action, 0, len(actions)),
	}
	if result.ProductName == "" {
		result.ProductName = result.ProductKey
	}
	for _, property := range properties {
		result.Properties = append(result.Properties, publicdevice.Property{
			ID:           property.ID,
			Identifier:   property.Identifier,
			Name:         property.Name,
			DataType:     property.DataType,
			Unit:         property.Unit,
			ReadWrite:    property.IsRw,
			Precision:    property.Precision,
			MinValue:     property.MinValue,
			MaxValue:     property.MaxValue,
			EnumValues:   property.EnumValues,
			DefaultValue: property.DefaultValue,
		})
	}
	for _, action := range actions {
		result.Actions = append(result.Actions, publicdevice.Action{
			ID:               action.ID,
			Identifier:       action.Identifier,
			Name:             action.Name,
			InputSchema:      action.InputSchema,
			TimeoutMs:        action.TimeoutMs,
			ReadbackRequired: action.ReadbackRequired,
			Available:        availableActions[action.ID],
		})
	}
	return result, nil
}

func (r *ChannelSpecResolver) availableActionIDs(ctx context.Context, device data.Devices) (map[uint]bool, error) {
	if device.ProfileID <= 0 {
		return map[uint]bool{}, nil
	}
	var pointIDs []int64
	if err := r.db.WithContext(ctx).
		Table("device_point_bindings AS dpb").
		Select("ppb.point_id").
		Joins("JOIN product_point_bindings AS ppb ON ppb.id = dpb.product_point_binding_id").
		Where(
			"dpb.device_id = ? AND dpb.enabled = ? AND dpb.deleted_at IS NULL "+
				"AND ppb.enabled = ? AND ppb.point_kind = ? AND ppb.profile_id = ? AND ppb.deleted_at IS NULL",
			device.ID,
			true,
			true,
			data.PointKindAction,
			device.ProfileID,
		).
		Distinct().
		Pluck("ppb.point_id", &pointIDs).Error; err != nil {
		return nil, fmt.Errorf("load available mobile actions: %w", err)
	}
	available := make(map[uint]bool, len(pointIDs))
	for _, pointID := range pointIDs {
		if pointID > 0 {
			available[uint(pointID)] = true
		}
	}
	return available, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
