package data

import "gorm.io/gorm"

const (
	ProtocolProfileStatusDraft    = "draft"
	ProtocolProfileStatusEnabled  = "enabled"
	ProtocolProfileStatusDisabled = "disabled"
)

// ProductProtocolProfile defines how a product is accessed through one protocol version.
type ProductProtocolProfile struct {
	gorm.Model
	ProductID      int64  `json:"product_id" gorm:"column:product_id;not null;index:idx_product_protocol_profiles_product_id;uniqueIndex:uk_product_protocol_profile_version,priority:1"`
	ProtocolType   string `json:"protocol_type" gorm:"column:protocol_type;not null;size:32;index:idx_product_protocol_profiles_protocol_type"`
	DriverName     string `json:"driver_name" gorm:"column:driver_name;not null;size:64"`
	ProfileVersion string `json:"profile_version" gorm:"column:profile_version;not null;size:64;uniqueIndex:uk_product_protocol_profile_version,priority:2"`
	ConfigSchema   string `json:"config_schema" gorm:"column:config_schema;type:text"`
	Status         string `json:"status" gorm:"column:status;not null;size:16;default:draft;index:idx_product_protocol_profiles_status"`
}

func (m *ProductProtocolProfile) TableName() string {
	return "product_protocol_profiles"
}

type ProductProtocolProfileRepo interface {
	Add(profile ProductProtocolProfile) error
	Update(profile ProductProtocolProfile) error
	Delete(id int64) error
	FindByID(id int64) (error, *ProductProtocolProfile)
	FindByProductID(productID int64) (error, []ProductProtocolProfile)
	FindByProductIDAndVersion(productID int64, version string) (error, *ProductProtocolProfile)
	FindEnabledByProductID(productID int64) (error, []ProductProtocolProfile)
	FindAll() (error, []ProductProtocolProfile)
	UpdateStatus(id int64, status string) error
}
