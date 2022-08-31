package data

type KvProperty struct {
	Id            int64  `json:"id" gorm:"column:id"`                           //
	Name          string `json:"name" gorm:"column:name"`                       // 属性名称
	Identifier    string `json:"identifier" gorm:"column:identifier"`           // 属性标识符
	DataType      string `json:"dataType" gorm:"column:dataType"`               // 属性数据类型
	Unit          string `json:"unit" gorm:"column:unit"`                       // 属性单位
	IsRw          string `json:"is_rw" gorm:"column:is_rw"`                     // 是否可读写(r,w,rw)
	SubProperty   uint16 `json:"sub_property" gorm:"column:sub_property"`       // 是否有子属性
	SubPropertyId int64  `json:"sub_property_id" gorm:"column:sub_property_id"` // 属性id
}
