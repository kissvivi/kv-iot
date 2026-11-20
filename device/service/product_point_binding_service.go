package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"kv-iot/device/data"
	"strconv"
	"strings"

	modbusdriver "kv-iot/datacenter/protocol/modbus"

	"gorm.io/gorm"
)

type ProductPointBindingUpdate struct {
	PointKind      *string `json:"point_kind"`
	PointID        *int64  `json:"point_id"`
	CollectionMode *string `json:"collection_mode"`
	PollGroup      *string `json:"poll_group"`
	BindingConfig  *string `json:"binding_config"`
	Enabled        *bool   `json:"enabled"`
}

type BindingValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type ProductPointBindingService interface {
	Create(profileID int64, binding data.ProductPointBinding) (*data.ProductPointBinding, error)
	GetByID(id int64) (*data.ProductPointBinding, error)
	ListByProfile(profileID int64) ([]data.ProductPointBinding, error)
	Update(id int64, update ProductPointBindingUpdate) (*data.ProductPointBinding, error)
	Delete(id int64) error
	Validate(protocolType string, bindingConfig string) BindingValidationResult
}

type ProductPointBindingServiceImpl struct {
	repo     data.ProductPointBindingRepo
	profiles ProductProtocolProfileService
	models   ProductModelService
}

func NewProductPointBindingService(
	repo data.ProductPointBindingRepo,
	profiles ProductProtocolProfileService,
	models ProductModelService,
) *ProductPointBindingServiceImpl {
	return &ProductPointBindingServiceImpl{repo: repo, profiles: profiles, models: models}
}

func (s *ProductPointBindingServiceImpl) Create(profileID int64, binding data.ProductPointBinding) (*data.ProductPointBinding, error) {
	binding.ID = 0
	binding.ProfileID = profileID
	if err := s.validate(&binding); err != nil {
		return nil, err
	}
	if err := s.repo.Add(binding); err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *ProductPointBindingServiceImpl) GetByID(id int64) (*data.ProductPointBinding, error) {
	err, binding := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *ProductPointBindingServiceImpl) ListByProfile(profileID int64) ([]data.ProductPointBinding, error) {
	if _, err := s.profiles.GetByID(profileID); err != nil {
		return nil, err
	}
	err, bindings := s.repo.FindByProfileID(profileID)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (s *ProductPointBindingServiceImpl) Update(id int64, update ProductPointBindingUpdate) (*data.ProductPointBinding, error) {
	binding, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if update.PointKind != nil {
		binding.PointKind = *update.PointKind
	}
	if update.PointID != nil {
		binding.PointID = *update.PointID
	}
	if update.CollectionMode != nil {
		binding.CollectionMode = *update.CollectionMode
	}
	if update.PollGroup != nil {
		binding.PollGroup = *update.PollGroup
	}
	if update.BindingConfig != nil {
		binding.BindingConfig = *update.BindingConfig
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

func (s *ProductPointBindingServiceImpl) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ProductPointBindingServiceImpl) Validate(protocolType string, bindingConfig string) BindingValidationResult {
	errs := validateBindingConfig(protocolType, bindingConfig)
	return BindingValidationResult{Valid: len(errs) == 0, Errors: errs}
}

func (s *ProductPointBindingServiceImpl) validate(binding *data.ProductPointBinding) error {
	if binding.ProfileID <= 0 {
		return errors.New("profile_id is required")
	}
	profile, err := s.profiles.GetByID(binding.ProfileID)
	if err != nil {
		return err
	}

	binding.PointKind = strings.TrimSpace(binding.PointKind)
	if !validPointKind(binding.PointKind) {
		return fmt.Errorf("invalid point_kind %q", binding.PointKind)
	}
	if binding.PointID <= 0 {
		return errors.New("point_id is required")
	}
	binding.CollectionMode = strings.TrimSpace(binding.CollectionMode)
	if binding.CollectionMode == "" {
		if binding.PointKind == data.PointKindProperty {
			binding.CollectionMode = data.CollectionModePoll
		} else {
			binding.CollectionMode = data.CollectionModeOnDemand
		}
	}
	if !validCollectionMode(binding.CollectionMode) {
		return fmt.Errorf("invalid collection_mode %q", binding.CollectionMode)
	}
	if profile.ProtocolType == data.ChannelTypeModbusTCP &&
		binding.CollectionMode == data.CollectionModeStream {
		return errors.New("modbus_tcp does not support stream collection mode")
	}
	binding.PollGroup = strings.TrimSpace(binding.PollGroup)
	if binding.PollGroup != "" && binding.CollectionMode != data.CollectionModePoll {
		return errors.New("poll_group can only be used with poll collection mode")
	}

	if err := s.validatePoint(binding, profile.ProductID); err != nil {
		return err
	}
	if errs := validateBindingConfig(profile.ProtocolType, binding.BindingConfig); len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	err, existing := s.repo.FindByProfileIDAndPoint(binding.ProfileID, binding.PointKind, binding.PointID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && existing != nil && existing.ID != binding.ID {
		return errors.New("point binding already exists for profile point")
	}
	return nil
}

func (s *ProductPointBindingServiceImpl) validatePoint(binding *data.ProductPointBinding, productID int64) error {
	err, model := s.models.GetProductModel(productID)
	if err != nil {
		return err
	}
	switch binding.PointKind {
	case data.PointKindProperty:
		for _, point := range model.Properties {
			if int64(point.ID) == binding.PointID {
				return nil
			}
		}
	case data.PointKindAction:
		for _, point := range model.Actions {
			if int64(point.ID) == binding.PointID {
				return nil
			}
		}
	case data.PointKindEvent:
		for _, point := range model.Events {
			if int64(point.ID) == binding.PointID {
				return nil
			}
		}
	}
	return fmt.Errorf("%s point %d does not belong to product %d", binding.PointKind, binding.PointID, productID)
}

func validPointKind(kind string) bool {
	switch kind {
	case data.PointKindProperty, data.PointKindAction, data.PointKindEvent:
		return true
	default:
		return false
	}
}

func validCollectionMode(mode string) bool {
	switch mode {
	case data.CollectionModePoll, data.CollectionModeStream, data.CollectionModeOnDemand:
		return true
	default:
		return false
	}
}

func validateBindingConfig(protocolType string, bindingConfig string) []string {
	config, err := parseJSONObject(bindingConfig)
	if err != nil {
		return []string{"binding_config must be a JSON object: " + err.Error()}
	}

	var errs []string
	switch protocolType {
	case data.ChannelTypeMQTT:
		direction, _ := config["direction"].(string)
		if direction != "uplink" && direction != "downlink" {
			errs = append(errs, "mqtt direction must be uplink or downlink")
		}
		topic, _ := config["topicTemplate"].(string)
		if strings.TrimSpace(topic) == "" {
			errs = append(errs, "mqtt topicTemplate is required")
		}
		if qos, ok := config["qos"]; ok && !validQoS(qos) {
			errs = append(errs, "mqtt qos must be 0, 1 or 2")
		}
	case data.ChannelTypeModbusTCP:
		raw, err := json.Marshal(config)
		if err != nil {
			errs = append(errs, "modbus binding config cannot be encoded")
			break
		}
		if err := modbusdriver.ValidatePointBinding(raw); err != nil {
			errs = append(errs, err.Error())
		}
	case "tcp_raw":
		if mode, ok := config["mode"].(string); ok && mode != "raw" {
			errs = append(errs, "tcp_raw mode must be raw")
		}
	case "tcp_frame":
		if _, ok := config["framing"].(map[string]interface{}); !ok {
			errs = append(errs, "tcp_frame framing object is required")
		}
	default:
		errs = append(errs, "unsupported protocol_type "+protocolType)
	}
	return errs
}

func validQoS(value interface{}) bool {
	switch v := value.(type) {
	case json.Number:
		n, err := strconv.Atoi(v.String())
		return err == nil && n >= 0 && n <= 2
	case float64:
		return v == 0 || v == 1 || v == 2
	case int:
		return v >= 0 && v <= 2
	default:
		return false
	}
}

func validModbusArea(area string) bool {
	switch area {
	case "coil", "discrete_input", "holding_register", "input_register":
		return true
	default:
		return false
	}
}

func validNonNegativeNumber(value interface{}) bool {
	switch v := value.(type) {
	case json.Number:
		n, err := strconv.ParseInt(v.String(), 10, 64)
		return err == nil && n >= 0
	case float64:
		return v >= 0 && v == float64(int64(v))
	case int:
		return v >= 0
	default:
		return false
	}
}
