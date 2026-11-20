package service

import (
	"errors"
	"fmt"
	"kv-iot/device/data"

	"gorm.io/gorm"
)

type DevicePointBindingUpdate struct {
	ChannelInstanceID     *int64  `json:"channel_instance_id"`
	ProductPointBindingID *int64  `json:"product_point_binding_id"`
	BindingOverride       *string `json:"binding_override"`
	Enabled               *bool   `json:"enabled"`
}

type DevicePointBindingService interface {
	Create(deviceID int64, binding data.DevicePointBinding) (*data.DevicePointBinding, error)
	GetByID(id int64) (*data.DevicePointBinding, error)
	ListByDevice(deviceID int64) ([]data.DevicePointBinding, error)
	Update(id int64, update DevicePointBindingUpdate) (*data.DevicePointBinding, error)
	Delete(id int64) error
	SyncForDevice(deviceID int64) ([]data.DevicePointBinding, error)
}

type DevicePointBindingServiceImpl struct {
	repo             data.DevicePointBindingRepo
	devices          DevicesService
	productBindings  ProductPointBindingService
	channelInstances ChannelInstanceService
}

func NewDevicePointBindingService(
	repo data.DevicePointBindingRepo,
	devices DevicesService,
	productBindings ProductPointBindingService,
	channelInstances ChannelInstanceService,
) *DevicePointBindingServiceImpl {
	return &DevicePointBindingServiceImpl{
		repo:             repo,
		devices:          devices,
		productBindings:  productBindings,
		channelInstances: channelInstances,
	}
}

func (s *DevicePointBindingServiceImpl) Create(deviceID int64, binding data.DevicePointBinding) (*data.DevicePointBinding, error) {
	binding.ID = 0
	binding.DeviceID = deviceID
	if err := s.validate(&binding); err != nil {
		return nil, err
	}
	if err := s.repo.Add(binding); err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *DevicePointBindingServiceImpl) GetByID(id int64) (*data.DevicePointBinding, error) {
	err, binding := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *DevicePointBindingServiceImpl) ListByDevice(deviceID int64) ([]data.DevicePointBinding, error) {
	if _, err := s.devices.GetDeviceByID(deviceID); err != nil {
		return nil, err
	}
	err, bindings := s.repo.FindByDeviceID(deviceID)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (s *DevicePointBindingServiceImpl) Update(id int64, update DevicePointBindingUpdate) (*data.DevicePointBinding, error) {
	binding, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if update.ChannelInstanceID != nil {
		binding.ChannelInstanceID = *update.ChannelInstanceID
	}
	if update.ProductPointBindingID != nil {
		binding.ProductPointBindingID = *update.ProductPointBindingID
	}
	if update.BindingOverride != nil {
		binding.BindingOverride = *update.BindingOverride
	}
	if update.Enabled != nil {
		binding.Enabled = *update.Enabled
	}
	if err := s.validate(binding); err != nil {
		return nil, err
	}
	if err := s.repo.Update(*binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *DevicePointBindingServiceImpl) Delete(id int64) error {
	return s.repo.Delete(id)
}

// SyncForDevice creates missing device bindings from the enabled product
// bindings of the device's selected protocol profile. Existing bindings and
// their overrides are preserved.
func (s *DevicePointBindingServiceImpl) SyncForDevice(deviceID int64) ([]data.DevicePointBinding, error) {
	device, err := s.devices.GetDeviceByID(deviceID)
	if err != nil {
		return nil, err
	}
	if device.ProfileID <= 0 || device.ChannelInstanceID <= 0 {
		return nil, nil
	}
	productBindings, err := s.productBindings.ListByProfile(device.ProfileID)
	if err != nil {
		return nil, err
	}
	err, existing := s.repo.FindByDeviceID(deviceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	created := make([]data.DevicePointBinding, 0)
	for _, productBinding := range missingProductBindings(
		productBindings,
		existing,
		device.ChannelInstanceID,
	) {
		productBindingID := int64(productBinding.ID)
		binding := data.DevicePointBinding{
			DeviceID:              deviceID,
			ChannelInstanceID:     device.ChannelInstanceID,
			ProductPointBindingID: productBindingID,
			BindingOverride:       "{}",
			Enabled:               true,
		}
		createdBinding, err := s.Create(deviceID, binding)
		if err != nil {
			return created, err
		}
		created = append(created, *createdBinding)
	}
	return created, nil
}

func missingProductBindings(
	productBindings []data.ProductPointBinding,
	existing []data.DevicePointBinding,
	channelInstanceID int64,
) []data.ProductPointBinding {
	existingIDs := make(map[int64]struct{}, len(existing))
	for _, binding := range existing {
		if binding.ChannelInstanceID == channelInstanceID {
			existingIDs[binding.ProductPointBindingID] = struct{}{}
		}
	}
	missing := make([]data.ProductPointBinding, 0)
	for _, productBinding := range productBindings {
		if !productBinding.Enabled {
			continue
		}
		if _, exists := existingIDs[int64(productBinding.ID)]; exists {
			continue
		}
		missing = append(missing, productBinding)
	}
	return missing
}

func (s *DevicePointBindingServiceImpl) validate(binding *data.DevicePointBinding) error {
	if binding.DeviceID <= 0 {
		return errors.New("device_id is required")
	}
	device, err := s.devices.GetDeviceByID(binding.DeviceID)
	if err != nil {
		return err
	}
	if device.ProfileID <= 0 {
		return errors.New("device must select a protocol profile before binding points")
	}
	if binding.ProductPointBindingID <= 0 {
		return errors.New("product_point_binding_id is required")
	}
	productBinding, err := s.productBindings.GetByID(binding.ProductPointBindingID)
	if err != nil {
		return err
	}
	if productBinding.ProfileID != device.ProfileID {
		return errors.New("product_point_binding_id does not belong to device protocol profile")
	}

	if binding.ChannelInstanceID == 0 {
		binding.ChannelInstanceID = device.ChannelInstanceID
	}
	if binding.ChannelInstanceID <= 0 {
		return errors.New("channel_instance_id is required")
	}
	if s.channelInstances != nil {
		err, channel := s.channelInstances.GetChannelInstanceByID(binding.ChannelInstanceID)
		if err != nil {
			return err
		}
		if channel == nil {
			return fmt.Errorf("channel instance %d not found", binding.ChannelInstanceID)
		}
	}

	if binding.BindingOverride == "" {
		binding.BindingOverride = "{}"
	}
	if _, err := parseJSONObject(binding.BindingOverride); err != nil {
		return fmt.Errorf("binding_override must be a JSON object: %w", err)
	}

	err, existing := s.repo.FindByInstance(binding.DeviceID, binding.ChannelInstanceID, binding.ProductPointBindingID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && existing != nil && existing.ID != binding.ID {
		return errors.New("device point binding already exists")
	}
	return nil
}
