package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kv-iot/device/data"
	"strings"

	"gorm.io/gorm"
)

var supportedProtocolTypes = map[string]struct{}{
	data.ChannelTypeMQTT:      {},
	data.ChannelTypeModbusTCP: {},
	"tcp_raw":                 {},
	"tcp_frame":               {},
}

type ProductReader interface {
	GetProductByID(id int64) (*data.Products, error)
}

type ProductProtocolProfileUpdate struct {
	ProtocolType   *string `json:"protocol_type"`
	DriverName     *string `json:"driver_name"`
	ProfileVersion *string `json:"profile_version"`
	ConfigSchema   *string `json:"config_schema"`
	Status         *string `json:"status"`
}

type ProductProtocolProfileService interface {
	Create(productID int64, profile data.ProductProtocolProfile) (*data.ProductProtocolProfile, error)
	GetByID(id int64) (*data.ProductProtocolProfile, error)
	ListByProduct(productID int64) ([]data.ProductProtocolProfile, error)
	Update(id int64, update ProductProtocolProfileUpdate) (*data.ProductProtocolProfile, error)
	Delete(id int64) error
}

type ProductProtocolProfileServiceImpl struct {
	repo     data.ProductProtocolProfileRepo
	products ProductReader
}

func NewProductProtocolProfileService(repo data.ProductProtocolProfileRepo, products ProductReader) *ProductProtocolProfileServiceImpl {
	return &ProductProtocolProfileServiceImpl{repo: repo, products: products}
}

func (s *ProductProtocolProfileServiceImpl) Create(productID int64, profile data.ProductProtocolProfile) (*data.ProductProtocolProfile, error) {
	if profile.ProductID != 0 && profile.ProductID != productID {
		return nil, errors.New("product_id in body must match path")
	}
	profile.ID = 0
	profile.ProductID = productID
	if err := s.validate(&profile); err != nil {
		return nil, err
	}
	if err := s.repo.Add(profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ProductProtocolProfileServiceImpl) GetByID(id int64) (*data.ProductProtocolProfile, error) {
	err, profile := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *ProductProtocolProfileServiceImpl) ListByProduct(productID int64) ([]data.ProductProtocolProfile, error) {
	if s.products != nil {
		if _, err := s.products.GetProductByID(productID); err != nil {
			return nil, err
		}
	}
	err, profiles := s.repo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *ProductProtocolProfileServiceImpl) Update(id int64, update ProductProtocolProfileUpdate) (*data.ProductProtocolProfile, error) {
	profile, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if update.ProtocolType != nil {
		profile.ProtocolType = *update.ProtocolType
	}
	if update.DriverName != nil {
		profile.DriverName = *update.DriverName
	}
	if update.ProfileVersion != nil {
		profile.ProfileVersion = *update.ProfileVersion
	}
	if update.ConfigSchema != nil {
		profile.ConfigSchema = *update.ConfigSchema
	}
	if update.Status != nil {
		profile.Status = *update.Status
	}
	if err := s.validate(profile); err != nil {
		return nil, err
	}
	if err := s.repo.Update(*profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *ProductProtocolProfileServiceImpl) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ProductProtocolProfileServiceImpl) validate(profile *data.ProductProtocolProfile) error {
	if profile.ProductID <= 0 {
		return errors.New("product_id is required")
	}
	if s.products != nil {
		if _, err := s.products.GetProductByID(profile.ProductID); err != nil {
			return fmt.Errorf("product not found: %w", err)
		}
	}

	profile.ProtocolType = strings.TrimSpace(profile.ProtocolType)
	if _, ok := supportedProtocolTypes[profile.ProtocolType]; !ok {
		return fmt.Errorf("unsupported protocol_type %q", profile.ProtocolType)
	}
	profile.DriverName = strings.TrimSpace(profile.DriverName)
	if profile.DriverName == "" {
		profile.DriverName = profile.ProtocolType
	}
	profile.ProfileVersion = strings.TrimSpace(profile.ProfileVersion)
	if profile.ProfileVersion == "" {
		return errors.New("profile_version is required")
	}
	profile.Status = strings.TrimSpace(profile.Status)
	if profile.Status == "" {
		profile.Status = data.ProtocolProfileStatusDraft
	}
	if !validProfileStatus(profile.Status) {
		return fmt.Errorf("invalid profile status %q", profile.Status)
	}
	if strings.TrimSpace(profile.ConfigSchema) != "" {
		if _, err := parseJSONObject(profile.ConfigSchema); err != nil {
			return fmt.Errorf("config_schema must be a JSON object: %w", err)
		}
	}

	err, existing := s.repo.FindByProductIDAndVersion(profile.ProductID, profile.ProfileVersion)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && existing != nil && existing.ID != profile.ID {
		return errors.New("profile_version already exists for product")
	}
	return nil
}

func validProfileStatus(status string) bool {
	switch status {
	case data.ProtocolProfileStatusDraft, data.ProtocolProfileStatusEnabled, data.ProtocolProfileStatusDisabled:
		return true
	default:
		return false
	}
}

func parseJSONObject(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("JSON is empty")
	}
	var value map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("JSON object is required")
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("unexpected trailing JSON")
		}
		return nil, err
	}
	return value, nil
}
