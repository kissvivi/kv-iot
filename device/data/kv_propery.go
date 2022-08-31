package data

type KvProperty struct {
	ID            int64  `json:"id" gorm:"column:id"`
	Name          string `json:"name" gorm:"column:name"`                       // 属性名称
	Identifier    string `json:"identifier" gorm:"column:identifier"`           // 属性标识符
	DataType      string `json:"dataType" gorm:"column:dataType"`               // 属性数据类型
	Unit          string `json:"unit" gorm:"column:unit"`                       // 属性单位
	IsRw          string `json:"is_rw" gorm:"column:is_rw"`                     // 是否可读写(r,w,rw)
	SubProperty   int16  `json:"sub_property" gorm:"column:sub_property"`       // 是否有子属性
	SubPropertyID int64  `json:"sub_property_id" gorm:"column:sub_property_id"` // 属性id
	ProductKey    string `json:"product_key" gorm:"column:product_key"`         // 产品标识
	ProductID     int64  `json:"product_id" gorm:"column:product_id"`           // 产品id
}

func (m *KvProperty) TableName() string {
	return "kv_property"
}
