package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

// ChannelHandler 存储通道处理器，负责处理通道的创建、更新、删除等请求
type ChannelHandler struct {
	channelService *service.ChannelService // 通道服务引用
}

// NewChannelHandler 创建ChannelHandler实例
func NewChannelHandler() *ChannelHandler {
	return &ChannelHandler{
		channelService: service.NewChannelService(),
	}
}

// Create 创建新的存储通道
// POST /api/v1/channel (需要admin权限)
func (h *ChannelHandler) Create(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Type      string                 `json:"type" binding:"required"`
		Config    map[string]interface{} `json:"config" binding:"required"`
		Weight    int                    `json:"weight"`
		Quota     model.QuotaConfig      `json:"quota"`
		RateLimit model.RateLimitConfig  `json:"rateLimit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("create channel: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if req.Weight <= 0 {
		req.Weight = 100
	}

	channel, err := h.channelService.CreateChannel(c.Request.Context(), req.Name, req.Type, req.Config, req.Weight, req.Quota, req.RateLimit)
	if err != nil {
		utils.Errorf("create channel: create failed, name=%s, type=%s, error=%v", req.Name, req.Type, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("create channel: success, channelID=%s, name=%s, type=%s", channel.ID, req.Name, req.Type)
	response.Success(c, channel)
}

func (h *ChannelHandler) Update(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("update channel: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Config    map[string]interface{} `json:"config" binding:"required"`
		Weight    int                    `json:"weight"`
		Quota     model.QuotaConfig      `json:"quota"`
		RateLimit model.RateLimitConfig  `json:"rateLimit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("update channel: invalid request, channelID=%s, error=%v", channelID, err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.channelService.UpdateChannel(c.Request.Context(), channelID, req.Name, req.Config, req.Weight, req.Quota, req.RateLimit); err != nil {
		utils.Errorf("update channel: update failed, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update channel: success, channelID=%s", channelID)
	response.Success(c, nil)
}

// TestChannel 测试渠道连接
// POST /api/v1/channel/:id/test (需要admin权限)
func (h *ChannelHandler) TestChannel(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("test channel: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	// 执行健康检查
	if err := h.channelService.HealthCheck(c.Request.Context(), channelID); err != nil {
		utils.Errorf("test channel: test failed, channelID=%s, error=%v", channelID, err)
		response.Success(c, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	utils.Infof("test channel: success, channelID=%s", channelID)
	response.Success(c, gin.H{
		"success": true,
		"message": "channel connection successful",
	})
}

// Delete 删除存储通道
// DELETE /api/v1/channel/:id (需要admin权限)
func (h *ChannelHandler) Delete(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("delete channel: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	if err := h.channelService.DeleteChannel(c.Request.Context(), channelID); err != nil {
		utils.Errorf("delete channel: delete failed, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete channel: success, channelID=%s", channelID)
	response.Success(c, nil)
}

// Get 获取存储通道详情
// GET /api/v1/channel/:id
func (h *ChannelHandler) Get(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("get channel: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	channel, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		utils.Errorf("get channel: channel not found, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrNotFound, "channel not found")
		return
	}

	// 解析通道配置
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		utils.Warnf("get channel: parse config failed, channelID=%s, error=%v", channelID, err)
	}

	response.Success(c, gin.H{
		"id":                channel.ID,
		"name":              channel.Name,
		"type":              channel.Type,
		"config":            config,
		"enabled":           channel.Enabled,
		"status":            channel.Status,
		"weight":            channel.Weight,
		"quotaEnabled":      channel.QuotaEnabled,
		"quotaLimit":        channel.QuotaLimit,
		"quotaThreshold":    channel.QuotaThreshold,
		"dailyUploadLimit":  channel.DailyUploadLimit,
		"hourlyUploadLimit": channel.HourlyUploadLimit,
		"minIntervalMs":     channel.MinIntervalMs,
		"createdAt":         channel.CreatedAt,
		"updatedAt":         channel.UpdatedAt,
	})
}

// List 获取所有存储通道列表
// GET /api/v1/channel
func (h *ChannelHandler) List(c *gin.Context) {
	channels, err := h.channelService.ListChannels(c.Request.Context())
	if err != nil {
		utils.Errorf("list channels: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 构建返回结果
	result := make([]gin.H, len(channels))
	for i, ch := range channels {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(ch.Config), &config); err != nil {
			utils.Warnf("list channels: parse config failed, channelID=%s, error=%v", ch.ID, err)
		}

		result[i] = gin.H{
			"id":                ch.ID,
			"name":              ch.Name,
			"type":              ch.Type,
			"config":            config,
			"enabled":           ch.Enabled,
			"status":            ch.Status,
			"weight":            ch.Weight,
			"usedSpace":         ch.UsedSpace,
			"quotaEnabled":      ch.QuotaEnabled,
			"quotaLimit":        ch.QuotaLimit,
			"quotaThreshold":    ch.QuotaThreshold,
			"dailyUploadLimit":  ch.DailyUploadLimit,
			"dailyUploads":      ch.DailyUploads,
			"hourlyUploadLimit": ch.HourlyUploadLimit,
			"hourlyUploads":     ch.HourlyUploads,
			"minIntervalMs":     ch.MinIntervalMs,
			"createdAt":         ch.CreatedAt,
			"updatedAt":         ch.UpdatedAt,
		}
	}

	response.Success(c, result)
}

// Enable 启用或禁用存储通道
// PUT /api/v1/channel/:id/enable (需要admin权限)
func (h *ChannelHandler) Enable(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("enable channel: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("enable channel: invalid request, channelID=%s, error=%v", channelID, err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.channelService.EnableChannel(c.Request.Context(), channelID, req.Enabled); err != nil {
		utils.Errorf("enable channel: update failed, channelID=%s, enabled=%v, error=%v", channelID, req.Enabled, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("enable channel: success, channelID=%s, enabled=%v", channelID, req.Enabled)
	response.Success(c, nil)
}

// GetStatus 获取通道状态
// GET /api/v1/channel/:id/status
func (h *ChannelHandler) GetStatus(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("get status: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	status, err := h.channelService.GetChannelStatus(c.Request.Context(), channelID)
	if err != nil {
		utils.Errorf("get status: channel not found, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrNotFound, "channel not found")
		return
	}

	response.Success(c, status)
}

// GetAllStatus 获取所有通道状态
// GET /api/v1/channels/status (需要认证)
func (h *ChannelHandler) GetAllStatus(c *gin.Context) {
	statuses, err := h.channelService.GetAllChannelStatus(c.Request.Context())
	if err != nil {
		utils.Errorf("get all status: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, statuses)
}

// GetChannelStats 获取渠道统计信息
// GET /api/v1/channel/:id/stats (需要认证)
func (h *ChannelHandler) GetChannelStats(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("get channel stats: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	type ChannelStat struct {
		ChannelID    string  `json:"channelId"`
		ChannelName  string  `json:"channelName"`
		ChannelType  string  `json:"channelType"`
		TotalUploads int64   `json:"totalUploads"`
		SuccessCount int64   `json:"successCount"`
		FailedCount  int64   `json:"failedCount"`
		SuccessRate  float64 `json:"successRate"`
		LastUsedAt   int64   `json:"lastUsedAt"`
	}

	var channel model.Channel
	if err := database.DB.First(&channel, "id = ?", channelID).Error; err != nil {
		utils.Errorf("get channel stats: channel not found, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrNotFound, "channel not found")
		return
	}

	var totalUploads int64
	var lastUsedAt time.Time

	database.DB.Model(&model.File{}).Where("channel_id = ?", channelID).Count(&totalUploads)
	database.DB.Model(&model.File{}).Where("channel_id = ?", channelID).Select("COALESCE(MAX(created_at), ?)", time.Time{}).Scan(&lastUsedAt)

	var successCount, failedCount int64
	database.DB.Table("file_accesses").
		Joins("JOIN files ON file_accesses.file_id = files.id").
		Where("files.channel_id = ? AND file_accesses.access_type = ?", channelID, "upload_success").
		Count(&successCount)
	database.DB.Table("file_accesses").
		Joins("JOIN files ON file_accesses.file_id = files.id").
		Where("files.channel_id = ? AND file_accesses.access_type = ?", channelID, "upload_failed").
		Count(&failedCount)

	var successRate float64
	totalAttempts := successCount + failedCount
	if totalAttempts > 0 {
		successRate = float64(successCount) / float64(totalAttempts) * 100
	} else if totalUploads > 0 {
		successRate = 100.0
	}

	stat := ChannelStat{
		ChannelID:    channel.ID,
		ChannelName:  channel.Name,
		ChannelType:  channel.Type,
		TotalUploads: totalUploads,
		SuccessCount: successCount,
		FailedCount:  failedCount,
		SuccessRate:  successRate,
		LastUsedAt:   lastUsedAt.Unix(),
	}

	utils.Infof("get channel stats: success, channelID=%s, totalUploads=%d, successRate=%.2f", channelID, totalUploads, successRate)
	response.Success(c, stat)
}

// HealthCheck 检查单个通道健康状态
// GET /api/v1/channel/:id/health (需要admin权限)
func (h *ChannelHandler) HealthCheck(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("health check: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	if err := h.channelService.HealthCheck(c.Request.Context(), channelID); err != nil {
		utils.Errorf("health check: check failed, channelID=%s, error=%v", channelID, err)
		response.Success(c, gin.H{
			"healthy": false,
			"error":   err.Error(),
		})
		return
	}

	response.Success(c, gin.H{
		"healthy": true,
	})
}

// HealthCheckAll 检查所有通道健康状态
// POST /api/v1/channels/health-check (需要admin权限)
func (h *ChannelHandler) HealthCheckAll(c *gin.Context) {
	results := h.channelService.HealthCheckAll(c.Request.Context())

	statuses := make([]gin.H, 0)
	for id, err := range results {
		status := gin.H{
			"channelId": id,
			"healthy":   err == nil,
		}
		if err != nil {
			status["error"] = err.Error()
		}
		statuses = append(statuses, status)
	}

	response.Success(c, statuses)
}

// SetWeight 设置通道权重
// PUT /api/v1/channel/:id/weight (需要admin权限)
func (h *ChannelHandler) SetWeight(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		utils.Warnf("set weight: channel id is required")
		response.ValidationError(c, "channel id is required")
		return
	}

	var req struct {
		Weight int `json:"weight" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("set weight: invalid request, channelID=%s, error=%v", channelID, err)
		response.ValidationError(c, "invalid request")
		return
	}

	// 设置通道权重
	if err := h.channelService.SetChannelWeight(channelID, req.Weight); err != nil {
		utils.Errorf("set weight: failed, channelID=%s, weight=%d, error=%v", channelID, req.Weight, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("set weight: success, channelID=%s, weight=%d", channelID, req.Weight)
	response.Success(c, nil)
}
