package service

import (
	"kv-iot/device/data"
)

// ProductModelService 产品物模型服务接口
type ProductModelService interface {
	// GetProductModel 获取完整产品物模型
	GetProductModel(productID int64) (error, *ProductModel)
	// CreateProductModel 创建产品物模型
	CreateProductModel(productID int64, model ProductModel) error
	// UpdateProductModel 更新产品物模型
	UpdateProductModel(productID int64, model ProductModel) error
	// ValidateDeviceData 验证设备数据是否符合物模型
	ValidateDeviceData(productID int64, data map[string]interface{}) (bool, error)
	// GetProductSchema 获取产品物模型JSON Schema
	GetProductSchema(productID int64) (error, map[string]interface{})
}

// ProductModel 产品物模型完整结构
type ProductModel struct {
	Properties []data.KvProperty `json:"properties"` // 属性列表
	Actions    []data.KvAction   `json:"actions"`    // 动作列表
	Events     []data.KvEvent    `json:"events"`     // 事件列表
}

// ProductModelServiceImpl 产品物模型服务实现
type ProductModelServiceImpl struct {
	propertyService KvPropertyService
	actionService   KvActionService
	eventService    KvEventService
}

// NewProductModelServiceImpl 创建产品物模型服务实例
func NewProductModelServiceImpl(
	propertyService KvPropertyService,
	actionService KvActionService,
	eventService KvEventService,
) *ProductModelServiceImpl {
	return &ProductModelServiceImpl{
		propertyService: propertyService,
		actionService:   actionService,
		eventService:    eventService,
	}
}

// GetProductModel 获取完整产品物模型
func (s *ProductModelServiceImpl) GetProductModel(productID int64) (error, *ProductModel) {
	// 获取属性列表
	err, properties := s.propertyService.GetPropertyByProductID(productID)
	if err != nil {
		return err, nil
	}

	// 获取动作列表
	action := data.KvAction{ProductID: productID}
	err, actions := s.actionService.GetKvAction(action)
	if err != nil {
		return err, nil
	}

	// 获取事件列表
	event := data.KvEvent{ProductID: productID}
	err, events := s.eventService.GetKvEvent(event)
	if err != nil {
		return err, nil
	}

	return nil, &ProductModel{
		Properties: properties,
		Actions:    actions,
		Events:     events,
	}
}

// CreateProductModel 创建产品物模型
func (s *ProductModelServiceImpl) CreateProductModel(productID int64, model ProductModel) error {
	// 创建属性
	for i := range model.Properties {
		model.Properties[i].ProductID = productID
		if err := s.propertyService.AddKvProperty(model.Properties[i]); err != nil {
			return err
		}
	}

	// 创建动作
	for i := range model.Actions {
		model.Actions[i].ProductID = productID
		if err := s.actionService.AddKvAction(model.Actions[i]); err != nil {
			return err
		}
	}

	// 创建事件
	for i := range model.Events {
		model.Events[i].ProductID = productID
		if err := s.eventService.AddKvEvent(model.Events[i]); err != nil {
			return err
		}
	}

	return nil
}

// UpdateProductModel 更新产品物模型
func (s *ProductModelServiceImpl) UpdateProductModel(productID int64, model ProductModel) error {
	// 更新逻辑略，实际应用中需要实现增量更新或完全覆盖
	// 这里简单返回nil，后续可以扩展
	return nil
}

// ValidateDeviceData 验证设备数据是否符合物模型
func (s *ProductModelServiceImpl) ValidateDeviceData(productID int64, deviceData map[string]interface{}) (bool, error) {
	// 验证逻辑略，实际应用中需要根据物模型定义验证数据格式
	// 这里简单返回true，后续可以扩展
	return true, nil
}

// GetProductSchema 获取产品物模型JSON Schema
func (s *ProductModelServiceImpl) GetProductSchema(productID int64) (error, map[string]interface{}) {
	// 获取物模型
	err, model := s.GetProductModel(productID)
	if err != nil {
		return err, nil
	}

	// 构建JSON Schema
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"actions":    make(map[string]interface{}),
		"events":     make(map[string]interface{}),
	}

	// 添加属性schema
	propertiesSchema := schema["properties"].(map[string]interface{})
	for _, prop := range model.Properties {
		propertiesSchema[prop.Identifier] = map[string]interface{}{
			"type":        prop.DataType,
			"description": prop.Name,
			"required":    false, // 使用默认值代替不存在的IsRequired字段
		}
	}

	// 添加动作schema
	actionsSchema := schema["actions"].(map[string]interface{})
	for _, action := range model.Actions {
		actionsSchema[action.Identifier] = map[string]interface{}{
			"description": action.Name,
		}
	}

	// 添加事件schema
	eventsSchema := schema["events"].(map[string]interface{})
	for _, event := range model.Events {
		eventsSchema[event.Identifier] = map[string]interface{}{
			"description": event.Name,
		}
	}

	return nil, schema
}