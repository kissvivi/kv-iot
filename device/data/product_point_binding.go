package data

import "gorm.io/gorm"

const (
	PointKindProperty = "property"
	PointKindAction   = "action"
	PointKindEvent    = "event"

	CollectionModePoll     = "poll"
	CollectionModeStream   = "stream"
	CollectionModeOnDemand = "on_demand"
)

// ProductPointBinding maps a product point to a protocol-specific access method.
type ProductPointBinding struct {
	gorm.Model
	ProfileID      int64  `json:"profile_id" gorm:"column:profile_id;not null;index:idx_product_point_bindings_profile_id;uniqueIndex:uk_product_point_binding_point,priority:1"`
	PointKind      string `json:"point_kind" gorm:"column:point_kind;not null;size:16;uniqueIndex:uk_product_point_binding_point,priority:2"`
	PointID        int64  `json:"point_id" gorm:"column:point_id;not null;uniqueIndex:uk_product_point_binding_point,priority:3"`
	CollectionMode string `json:"collection_mode" gorm:"column:collection_mode;not null;size:16;default:poll;index:idx_product_point_bindings_collection_mode"`
	PollGroup      string `json:"poll_group" gorm:"column:poll_group;size:64;index:idx_product_point_bindings_poll_group"`
	BindingConfig  string `json:"binding_config" gorm:"column:binding_config;type:text"`
	Enabled        bool   `json:"enabled" gorm:"column:enabled;not null;index:idx_product_point_bindings_enabled"`
}

func (m *ProductPointBinding) TableName() string {
	return "product_point_bindings"
}

type ProductPointBindingRepo interface {
	Add(binding ProductPointBinding) error
	Update(binding ProductPointBinding) error
	Delete(id int64) error
	FindByID(id int64) (error, *ProductPointBinding)
	FindByProfileID(profileID int64) (error, []ProductPointBinding)
	FindByProfileIDAndPointKind(profileID int64, pointKind string) (error, []ProductPointBinding)
	FindByProfileIDAndPoint(profileID int64, pointKind string, pointID int64) (error, *ProductPointBinding)
	FindAll() (error, []ProductPointBinding)
	UpdateEnabled(id int64, enabled bool) error
}
