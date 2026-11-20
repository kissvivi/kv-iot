package data

import "gorm.io/gorm"

type KvProperty struct {
	//ID            int64  `json:"id" gorm:"column:id"`
	gorm.Model
	Name          string   `json:"name" gorm:"column:name"`                       // 属性名称
	Identifier    string   `json:"identifier" gorm:"column:identifier"`           // 属性标识符
	DataType      string   `json:"dataType" gorm:"column:dataType"`               // 属性数据类型
	Unit          string   `json:"unit" gorm:"column:unit"`                       // 属性单位
	IsRw          string   `json:"is_rw" gorm:"column:is_rw"`                     // 是否可读写(r,w,rw)
	Required      bool     `json:"required" gorm:"column:required;default:false"` // 是否必需
	Nullable      bool     `json:"nullable" gorm:"column:nullable;default:true"`  // 是否允许空值
	MinValue      *float64 `json:"min_value" gorm:"column:min_value"`             // 最小值
	MaxValue      *float64 `json:"max_value" gorm:"column:max_value"`             // 最大值
	Precision     int      `json:"precision" gorm:"column:precision;default:0"`   // 精度
	EnumValues    string   `json:"enum_values" gorm:"column:enum_values;type:text"`
	DefaultValue  string   `json:"default_value" gorm:"column:default_value;type:text"`
	ReportMode    string   `json:"report_mode" gorm:"column:report_mode;size:32;default:change"`
	SubProperty   int16    `json:"sub_property" gorm:"column:sub_property"`       // 是否有子属性
	SubPropertyID int64    `json:"sub_property_id" gorm:"column:sub_property_id"` // 属性id
	ProductKey    string   `json:"product_key" gorm:"column:product_key"`         // 产品标识
	ProductID     int64    `json:"product_id" gorm:"column:product_id"`           // 产品id
}

func (m *KvProperty) TableName() string {
	return "kv_property"
}
