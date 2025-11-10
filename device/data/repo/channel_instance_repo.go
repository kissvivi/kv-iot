package repo

import (
	"kv-iot/device/data"
	"gorm.io/gorm"
)

// ChannelInstanceRepoImpl 通道实例仓库实现
type ChannelInstanceRepoImpl struct {
	db *gorm.DB
}

// NewChannelInstanceRepo 创建通道实例仓库
func NewChannelInstanceRepo(db *gorm.DB) data.ChannelInstanceRepo {
	return &ChannelInstanceRepoImpl{db: db}
}

// Add 添加通道实例
func (r *ChannelInstanceRepoImpl) Add(instance data.ChannelInstance) error {
	return r.db.Create(&instance).Error
}

// Update 更新通道实例
func (r *ChannelInstanceRepoImpl) Update(instance data.ChannelInstance) error {
	return r.db.Save(&instance).Error
}

// Delete 删除通道实例
func (r *ChannelInstanceRepoImpl) Delete(id int64) error {
	return r.db.Delete(&data.ChannelInstance{}, id).Error
}

// FindByID 根据ID查询通道实例
func (r *ChannelInstanceRepoImpl) FindByID(id int64) (error, *data.ChannelInstance) {
	var instance data.ChannelInstance
	err := r.db.First(&instance, id).Error
	if err != nil {
		return err, nil
	}
	return nil, &instance
}

// FindByProductID 根据产品ID查询通道实例
func (r *ChannelInstanceRepoImpl) FindByProductID(productID int64) (error, []data.ChannelInstance) {
	var instances []data.ChannelInstance
	err := r.db.Where("product_id = ?", productID).Find(&instances).Error
	return err, instances
}

// FindByProductKey 根据产品标识查询通道实例
func (r *ChannelInstanceRepoImpl) FindByProductKey(productKey string) (error, *data.ChannelInstance) {
	var instance data.ChannelInstance
	err := r.db.Where("product_key = ?", productKey).First(&instance).Error
	if err != nil {
		return err, nil
	}
	return nil, &instance
}

// FindByStatus 根据状态查询通道实例
func (r *ChannelInstanceRepoImpl) FindByStatus(status int) (error, []data.ChannelInstance) {
	var instances []data.ChannelInstance
	err := r.db.Where("status = ?", status).Find(&instances).Error
	return err, instances
}

// FindAll 查询所有通道实例
func (r *ChannelInstanceRepoImpl) FindAll() (error, []data.ChannelInstance) {
	var instances []data.ChannelInstance
	err := r.db.Find(&instances).Error
	return err, instances
}

// UpdateStatus 更新通道实例状态
func (r *ChannelInstanceRepoImpl) UpdateStatus(id int64, status int) error {
	return r.db.Model(&data.ChannelInstance{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateConnectionCount 更新连接计数
func (r *ChannelInstanceRepoImpl) UpdateConnectionCount(id int64, increment bool) error {
	query := r.db.Model(&data.ChannelInstance{}).Where("id = ?", id)
	if increment {
		return query.Update("current_conn", gorm.Expr("current_conn + ?", 1)).Error
	}
	return query.Update("current_conn", gorm.Expr("current_conn - ?", 1)).Error
}