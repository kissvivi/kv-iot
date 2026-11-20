package channel

import (
	"encoding/json"
	"fmt"
	"io"
	"kv-iot/config"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ApiChannelInstance 通道实例API处理器
type ApiChannelInstance struct {
	baseService *service.BaseService
}

// NewApiChannelInstance 创建通道实例API
func NewApiChannelInstance(baseService *service.BaseService) *ApiChannelInstance {
	return &ApiChannelInstance{baseService: baseService}
}

// CreateChannelInstance 创建通道实例
// @Summary 创建通道实例
// @Description 创建新的通道实例
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param instance body data.ChannelInstance true "通道实例信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance [post]
func (api *ApiChannelInstance) CreateChannelInstance(c *gin.Context) {
	var instance data.ChannelInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := api.baseService.ChannelInstanceService.CreateChannelInstance(&instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance created successfully", "data": instance})
}

// GetChannelInstance 获取通道实例
// @Summary 获取通道实例
// @Description 根据ID获取通道实例详情
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /channel/instance/{id} [get]
func (api *ApiChannelInstance) GetChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	err, instance := api.baseService.ChannelInstanceService.GetChannelInstanceByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel instance not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": instance})
}

// GetChannelInstances 获取通道实例列表
// @Summary 获取通道实例列表
// @Description 获取所有通道实例或根据产品ID筛选
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param product_id query int false "产品ID（可选）"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance [get]
func (api *ApiChannelInstance) GetChannelInstances(c *gin.Context) {
	productIDStr := c.Query("product_id")

	var instances []data.ChannelInstance
	var err error

	if productIDStr != "" {
		productID, err := strconv.ParseInt(productIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
			return
		}
		err, instances = api.baseService.ChannelInstanceService.GetChannelInstancesByProductID(productID)
	} else {
		err, instances = api.baseService.ChannelInstanceService.GetAllChannelInstances()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channel instances: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": instances})
}

// UpdateChannelInstance 更新通道实例
// @Summary 更新通道实例
// @Description 更新通道实例信息
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Param instance body data.ChannelInstance true "通道实例信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id} [put]
func (api *ApiChannelInstance) UpdateChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	var update service.ChannelInstanceUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if _, err := api.baseService.ChannelInstanceService.UpdateChannelInstance(id, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance updated successfully"})
}

// DeleteChannelInstance 删除通道实例
// @Summary 删除通道实例
// @Description 根据ID删除通道实例
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id} [delete]
func (api *ApiChannelInstance) DeleteChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	if err := api.baseService.ChannelInstanceService.DeleteChannelInstance(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance deleted successfully"})
}

// StartChannelInstance 启动通道实例
// @Summary 启动通道实例
// @Description 启动指定的通道实例
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id}/start [post]
func (api *ApiChannelInstance) StartChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	if err := forwardRuntimeAction(c, id, "start"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "启动运行通道失败: " + err.Error()})
		return
	}
	if err := api.baseService.ChannelInstanceService.StartChannelInstance(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance started successfully"})
}

// StopChannelInstance 停止通道实例
// @Summary 停止通道实例
// @Description 停止指定的通道实例
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id}/stop [post]
func (api *ApiChannelInstance) StopChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	if err := forwardRuntimeAction(c, id, "stop"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "停止运行通道失败: " + err.Error()})
		return
	}
	if err := api.baseService.ChannelInstanceService.StopChannelInstance(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance stopped successfully"})
}

// RestartChannelInstance 重启通道实例
// @Summary 重启通道实例
// @Description 重启指定的通道实例
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id}/restart [post]
func (api *ApiChannelInstance) RestartChannelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	if err := forwardRuntimeAction(c, id, "restart"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "重启运行通道失败: " + err.Error()})
		return
	}
	if err := api.baseService.ChannelInstanceService.RestartChannelInstance(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart channel instance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel instance restarted successfully"})
}

func forwardRuntimeAction(c *gin.Context, channelID int64, action string) error {
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		fmt.Sprintf("%s/channels/%d/%s", runtimeBaseURL(), channelID, action),
		nil,
	)
	if err != nil {
		return err
	}
	if authorization := c.GetHeader("Authorization"); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

func runtimeBaseURL() string {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("KV_RUNTIME_URL")), "/"); configured != "" {
		return configured
	}
	server := config.CONFIG.Application.DatacenterServer.HttpServer
	host := strings.TrimSpace(server.Host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(server.Port)
	if port == "" {
		port = "8102"
	}
	return "http://" + net.JoinHostPort(host, port) + "/runtime"
}

// GetChannelInstanceStatus 获取通道实例状态
// @Summary 获取通道实例状态
// @Description 获取指定通道实例的状态
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id}/status [get]
func (api *ApiChannelInstance) GetChannelInstanceStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	status, err := api.baseService.ChannelInstanceService.GetChannelInstanceStatus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channel instance status: " + err.Error()})
		return
	}

	// 获取状态文本
	statusText := api.getStatusText(status)

	c.JSON(http.StatusOK, gin.H{
		"status":     status,
		"statusText": statusText,
	})
}

// UpdateChannelConfig 更新通道配置
// @Summary 更新通道配置
// @Description 更新通道实例的配置
// @Tags 通道实例管理
// @Accept json
// @Produce json
// @Param id path int true "通道实例ID"
// @Param config body map[string]interface{} true "通道配置"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /channel/instance/{id}/config [put]
func (api *ApiChannelInstance) UpdateChannelConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel instance ID"})
		return
	}

	// 获取配置数据
	var configData map[string]interface{}
	if err := c.ShouldBindJSON(&configData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config data: " + err.Error()})
		return
	}

	// 将配置转换为JSON字符串
	configJSON, err := json.Marshal(configData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal config data: " + err.Error()})
		return
	}

	// 更新配置
	configText := string(configJSON)
	if _, err := api.baseService.ChannelInstanceService.UpdateChannelInstance(
		id,
		service.ChannelInstanceUpdate{Config: &configText},
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel config updated successfully"})
}

// getStatusText 获取状态文本描述
func (api *ApiChannelInstance) getStatusText(status int) string {
	switch status {
	case data.ChannelStatusDisabled:
		return "disabled"
	case data.ChannelStatusEnabled:
		return "enabled"
	case data.ChannelStatusRunning:
		return "running"
	case data.ChannelStatusError:
		return "error"
	default:
		return "unknown"
	}
}
