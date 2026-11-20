package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"kv-iot/db"
	"kv-iot/device/data"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// ProductModelService 产品物模型服务接口
type ProductModelService interface {
	// GetProductModel 获取完整产品物模型
	GetProductModel(productID int64) (error, *ProductModel)
	// CreateProductModel 创建产品物模型
	CreateProductModel(productID int64, model ProductModel) error
	// UpdateProductModel 更新产品物模型
	UpdateProductModel(productID int64, model ProductModel) error
	// DeleteProductModel 删除产品物模型
	DeleteProductModel(productID int64) error
	// ValidateDeviceData 验证设备数据是否符合物模型
	ValidateDeviceData(productID int64, data map[string]interface{}) (bool, error)
	// GetProductSchema 获取产品物模型JSON Schema
	GetProductSchema(productID int64) (error, map[string]interface{})
}

// ProductModel 产品物模型完整结构
type ProductModel struct {
	ProductID  int64             `json:"product_id"`
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
		ProductID:  productID,
		Properties: properties,
		Actions:    actions,
		Events:     events,
	}
}

// CreateProductModel 创建产品物模型
func (s *ProductModelServiceImpl) CreateProductModel(productID int64, model ProductModel) error {
	if err := validateProductModel(&model); err != nil {
		return err
	}
	product, err := loadProduct(productID)
	if err != nil {
		return err
	}

	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		for i := range model.Properties {
			model.Properties[i].ID = 0
			model.Properties[i].ProductID = productID
			model.Properties[i].ProductKey = product.ProductKey
			if err := tx.Create(&model.Properties[i]).Error; err != nil {
				return err
			}
		}
		for i := range model.Actions {
			model.Actions[i].ID = 0
			model.Actions[i].ProductID = productID
			model.Actions[i].ProductKey = product.ProductKey
			if err := tx.Create(&model.Actions[i]).Error; err != nil {
				return err
			}
		}
		for i := range model.Events {
			model.Events[i].ID = 0
			model.Events[i].ProductID = productID
			model.Events[i].ProductKey = product.ProductKey
			if err := tx.Create(&model.Events[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateProductModel 更新产品物模型
func (s *ProductModelServiceImpl) UpdateProductModel(productID int64, model ProductModel) error {
	if err := validateProductModel(&model); err != nil {
		return err
	}
	product, err := loadProduct(productID)
	if err != nil {
		return err
	}
	err, existing := s.GetProductModel(productID)
	if err != nil {
		return err
	}

	propertyByID, propertyByIdentifier := indexProperties(existing.Properties)
	actionByID, actionByIdentifier := indexActions(existing.Actions)
	eventByID, eventByIdentifier := indexEvents(existing.Events)

	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		propertyIDs := make([]uint, 0, len(model.Properties))
		for i := range model.Properties {
			property := &model.Properties[i]
			if property.ID > 0 && propertyByID[property.ID] != nil {
				property.ID = propertyByID[property.ID].ID
			} else if old := propertyByIdentifier[property.Identifier]; old != nil {
				property.ID = old.ID
			} else {
				property.ID = 0
			}
			property.ProductID = productID
			property.ProductKey = product.ProductKey
			if err := saveOrCreate(tx, property.ID, property); err != nil {
				return err
			}
			propertyIDs = append(propertyIDs, property.ID)
		}
		if err := deleteMissing(tx, &data.KvProperty{}, productID, propertyIDs); err != nil {
			return err
		}

		actionIDs := make([]uint, 0, len(model.Actions))
		for i := range model.Actions {
			action := &model.Actions[i]
			if action.ID > 0 && actionByID[action.ID] != nil {
				action.ID = actionByID[action.ID].ID
			} else if old := actionByIdentifier[action.Identifier]; old != nil {
				action.ID = old.ID
			} else {
				action.ID = 0
			}
			action.ProductID = productID
			action.ProductKey = product.ProductKey
			if err := saveOrCreate(tx, action.ID, action); err != nil {
				return err
			}
			actionIDs = append(actionIDs, action.ID)
		}
		if err := deleteMissing(tx, &data.KvAction{}, productID, actionIDs); err != nil {
			return err
		}

		eventIDs := make([]uint, 0, len(model.Events))
		for i := range model.Events {
			event := &model.Events[i]
			if event.ID > 0 && eventByID[event.ID] != nil {
				event.ID = eventByID[event.ID].ID
			} else if old := eventByIdentifier[event.Identifier]; old != nil {
				event.ID = old.ID
			} else {
				event.ID = 0
			}
			event.ProductID = productID
			event.ProductKey = product.ProductKey
			if err := saveOrCreate(tx, event.ID, event); err != nil {
				return err
			}
			eventIDs = append(eventIDs, event.ID)
		}
		return deleteMissing(tx, &data.KvEvent{}, productID, eventIDs)
	})
}

func saveOrCreate(tx *gorm.DB, id uint, value any) error {
	if id == 0 {
		return tx.Create(value).Error
	}
	return tx.Save(value).Error
}

func (s *ProductModelServiceImpl) DeleteProductModel(productID int64) error {
	if _, err := loadProduct(productID); err != nil {
		return err
	}
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", productID).Delete(&data.KvProperty{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", productID).Delete(&data.KvAction{}).Error; err != nil {
			return err
		}
		return tx.Where("product_id = ?", productID).Delete(&data.KvEvent{}).Error
	})
}

// ValidateDeviceData 验证设备数据是否符合物模型
func (s *ProductModelServiceImpl) ValidateDeviceData(productID int64, deviceData map[string]interface{}) (bool, error) {
	err, model := s.GetProductModel(productID)
	if err != nil {
		return false, err
	}
	for _, property := range model.Properties {
		value, exists := deviceData[property.Identifier]
		if !exists {
			if property.Required {
				return false, fmt.Errorf("property %s is required", property.Identifier)
			}
			continue
		}
		if value == nil {
			if !property.Nullable {
				return false, fmt.Errorf("property %s cannot be null", property.Identifier)
			}
			continue
		}
		if err := validatePropertyValue(property, value); err != nil {
			return false, err
		}
	}
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
		"required":   []string{},
	}

	// 添加属性schema
	propertiesSchema := schema["properties"].(map[string]interface{})
	required := make([]string, 0)
	for _, prop := range model.Properties {
		propertySchema := map[string]interface{}{
			"type":        jsonSchemaType(prop.DataType),
			"description": prop.Name,
			"nullable":    prop.Nullable,
		}
		if prop.MinValue != nil {
			propertySchema["minimum"] = *prop.MinValue
		}
		if prop.MaxValue != nil {
			propertySchema["maximum"] = *prop.MaxValue
		}
		propertiesSchema[prop.Identifier] = propertySchema
		if prop.Required {
			required = append(required, prop.Identifier)
		}
	}
	schema["required"] = required

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

func validateProductModel(model *ProductModel) error {
	propertyIdentifiers := make(map[string]struct{}, len(model.Properties))
	for i := range model.Properties {
		property := &model.Properties[i]
		property.Name = strings.TrimSpace(property.Name)
		property.Identifier = strings.TrimSpace(property.Identifier)
		if property.Name == "" || property.Identifier == "" {
			return errors.New("property name and identifier are required")
		}
		if _, exists := propertyIdentifiers[property.Identifier]; exists {
			return fmt.Errorf("duplicate property identifier %s", property.Identifier)
		}
		propertyIdentifiers[property.Identifier] = struct{}{}
		if strings.TrimSpace(property.DataType) == "" {
			return fmt.Errorf("property %s dataType is required", property.Identifier)
		}
		if property.IsRw == "" {
			property.IsRw = "r"
		}
		switch property.IsRw {
		case "r", "w", "rw":
		default:
			return fmt.Errorf("property %s is_rw is invalid", property.Identifier)
		}
		if property.Precision < 0 {
			return fmt.Errorf("property %s precision cannot be negative", property.Identifier)
		}
		if property.ReportMode == "" {
			property.ReportMode = "change"
		}
		switch property.ReportMode {
		case "change", "interval", "on_demand":
		default:
			return fmt.Errorf("property %s report_mode is invalid", property.Identifier)
		}
		if property.MinValue != nil && property.MaxValue != nil && *property.MinValue > *property.MaxValue {
			return fmt.Errorf("property %s min_value cannot exceed max_value", property.Identifier)
		}
		for field, raw := range map[string]string{
			"enum_values":   property.EnumValues,
			"default_value": property.DefaultValue,
		} {
			if strings.TrimSpace(raw) != "" && !json.Valid([]byte(raw)) {
				return fmt.Errorf("property %s %s must be valid JSON", property.Identifier, field)
			}
		}
	}

	actionIdentifiers := make(map[string]struct{}, len(model.Actions))
	for i := range model.Actions {
		action := &model.Actions[i]
		action.Name = strings.TrimSpace(action.Name)
		action.Identifier = strings.TrimSpace(action.Identifier)
		if action.Name == "" || action.Identifier == "" {
			return errors.New("action name and identifier are required")
		}
		if _, exists := actionIdentifiers[action.Identifier]; exists {
			return fmt.Errorf("duplicate action identifier %s", action.Identifier)
		}
		actionIdentifiers[action.Identifier] = struct{}{}
		if action.TimeoutMs < 0 {
			return fmt.Errorf("action %s timeout_ms cannot be negative", action.Identifier)
		}
		if action.TimeoutMs == 0 {
			action.TimeoutMs = 3000
		}
		for field, raw := range map[string]string{
			"input_schema":  action.InputSchema,
			"output_schema": action.OutputSchema,
		} {
			if strings.TrimSpace(raw) != "" && !json.Valid([]byte(raw)) {
				return fmt.Errorf("action %s %s must be valid JSON", action.Identifier, field)
			}
		}
	}

	eventIdentifiers := make(map[string]struct{}, len(model.Events))
	for i := range model.Events {
		event := &model.Events[i]
		event.Name = strings.TrimSpace(event.Name)
		event.Identifier = strings.TrimSpace(event.Identifier)
		if event.Name == "" || event.Identifier == "" {
			return errors.New("event name and identifier are required")
		}
		if _, exists := eventIdentifiers[event.Identifier]; exists {
			return fmt.Errorf("duplicate event identifier %s", event.Identifier)
		}
		eventIdentifiers[event.Identifier] = struct{}{}
		if event.Severity == "" {
			event.Severity = "info"
		}
		switch event.Severity {
		case "info", "warning", "error", "critical":
		default:
			return fmt.Errorf("event %s severity is invalid", event.Identifier)
		}
		if strings.TrimSpace(event.DataSchema) != "" && !json.Valid([]byte(event.DataSchema)) {
			return fmt.Errorf("event %s data_schema must be valid JSON", event.Identifier)
		}
	}
	return nil
}

func loadProduct(productID int64) (*data.Products, error) {
	if productID <= 0 {
		return nil, errors.New("product_id is required")
	}
	var product data.Products
	if err := db.MYSQLDB.First(&product, productID).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func deleteMissing(tx *gorm.DB, model interface{}, productID int64, keepIDs []uint) error {
	query := tx.Where("product_id = ?", productID)
	if len(keepIDs) > 0 {
		query = query.Where("id NOT IN ?", keepIDs)
	}
	return query.Delete(model).Error
}

func indexProperties(points []data.KvProperty) (map[uint]*data.KvProperty, map[string]*data.KvProperty) {
	byID := make(map[uint]*data.KvProperty, len(points))
	byIdentifier := make(map[string]*data.KvProperty, len(points))
	for i := range points {
		byID[points[i].ID] = &points[i]
		byIdentifier[points[i].Identifier] = &points[i]
	}
	return byID, byIdentifier
}

func indexActions(points []data.KvAction) (map[uint]*data.KvAction, map[string]*data.KvAction) {
	byID := make(map[uint]*data.KvAction, len(points))
	byIdentifier := make(map[string]*data.KvAction, len(points))
	for i := range points {
		byID[points[i].ID] = &points[i]
		byIdentifier[points[i].Identifier] = &points[i]
	}
	return byID, byIdentifier
}

func indexEvents(points []data.KvEvent) (map[uint]*data.KvEvent, map[string]*data.KvEvent) {
	byID := make(map[uint]*data.KvEvent, len(points))
	byIdentifier := make(map[string]*data.KvEvent, len(points))
	for i := range points {
		byID[points[i].ID] = &points[i]
		byIdentifier[points[i].Identifier] = &points[i]
	}
	return byID, byIdentifier
}

func validatePropertyValue(property data.KvProperty, value interface{}) error {
	switch strings.ToLower(property.DataType) {
	case "bool", "boolean":
		if reflect.TypeOf(value).Kind() != reflect.Bool {
			return fmt.Errorf("property %s must be boolean", property.Identifier)
		}
	case "int", "int16", "int32", "int64", "uint", "uint16", "uint32", "uint64":
		number, ok := numericValue(value)
		if !ok || number != float64(int64(number)) {
			return fmt.Errorf("property %s must be an integer", property.Identifier)
		}
		if property.MinValue != nil && number < *property.MinValue {
			return fmt.Errorf("property %s is below min_value", property.Identifier)
		}
		if property.MaxValue != nil && number > *property.MaxValue {
			return fmt.Errorf("property %s exceeds max_value", property.Identifier)
		}
	case "float", "float32", "float64", "double", "number":
		number, ok := numericValue(value)
		if !ok {
			return fmt.Errorf("property %s must be numeric", property.Identifier)
		}
		if property.MinValue != nil && number < *property.MinValue {
			return fmt.Errorf("property %s is below min_value", property.Identifier)
		}
		if property.MaxValue != nil && number > *property.MaxValue {
			return fmt.Errorf("property %s exceeds max_value", property.Identifier)
		}
	case "string", "text", "enum":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("property %s must be a string", property.Identifier)
		}
	}
	return nil
}

func numericValue(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func jsonSchemaType(dataType string) string {
	switch strings.ToLower(dataType) {
	case "bool", "boolean":
		return "boolean"
	case "int", "int16", "int32", "int64", "uint", "uint16", "uint32", "uint64":
		return "integer"
	case "float", "float32", "float64", "double", "number":
		return "number"
	case "string", "text", "enum":
		return "string"
	default:
		return dataType
	}
}
