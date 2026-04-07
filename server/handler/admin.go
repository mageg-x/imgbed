package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	fileService    *service.FileService
	channelService *service.ChannelService
	configService  *service.ConfigService
	tokenService   *service.TokenService
}

// NewAdminHandler 创建AdminHandler实例
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		fileService:    service.NewFileService(),
		channelService: service.NewChannelService(),
		configService:  service.NewConfigService(),
		tokenService:   service.NewTokenService(),
	}
}

// Dashboard 获取管理后台仪表盘数据
// GET /api/v1/admin/dashboard
func (h *AdminHandler) Dashboard(c *gin.Context) {
	var totalFiles int64
	var totalSize int64
	var totalChannels int64
	var enabledChannels int64

	database.DB.Model(&model.File{}).Count(&totalFiles)
	database.DB.Model(&model.File{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	database.DB.Model(&model.Channel{}).Count(&totalChannels)
	database.DB.Model(&model.Channel{}).Where("enabled = ?", true).Count(&enabledChannels)

	var todayUploads int64
	database.DB.Model(&model.File{}).Where("created_at >= date('now')").Count(&todayUploads)

	var todaySize int64
	database.DB.Model(&model.File{}).Where("created_at >= date('now')").Select("COALESCE(SUM(size), 0)").Scan(&todaySize)

	channelStatuses, _ := h.channelService.GetAllChannelStatus(c.Request.Context())

	response.Success(c, gin.H{
		"totalFiles":      totalFiles,
		"totalSize":       totalSize,
		"totalChannels":   totalChannels,
		"enabledChannels": enabledChannels,
		"todayUploads":    todayUploads,
		"todaySize":       todaySize,
		"channelStatuses": channelStatuses,
	})
}

// GetFiles 获取文件列表
// GET /api/v1/admin/files
func (h *AdminHandler) GetFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")
	channelID := c.Query("channel")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var files []model.File
	var total int64

	query := database.DB.Model(&model.File{})

	if search != "" {
		query = query.Where("name LIKE ? OR original_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		utils.Errorf("get files: query failed, page=%d, pageSize=%d, error=%v", page, pageSize, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 构建返回数据，直接使用 service 层的逻辑
	list, _, err := h.fileService.List(c.Request.Context(), page, pageSize, search, channelID, 0, 0, 0)
	if err != nil {
		utils.Errorf("get files: list failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// DeleteFile 删除单个文件
// DELETE /api/v1/admin/files/:id
func (h *AdminHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("delete file: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	if err := h.fileService.Delete(c.Request.Context(), fileID, ""); err != nil {
		utils.Errorf("delete file: delete failed, fileID=%s, error=%v", fileID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete file: success, fileID=%s", fileID)
	response.Success(c, nil)
}

// DeleteFiles 批量删除文件
// DELETE /api/v1/admin/files
func (h *AdminHandler) DeleteFiles(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("delete files: file ids are required")
		response.ValidationError(c, "file ids are required")
		return
	}

	success, failed, err := h.fileService.DeleteMultiple(c.Request.Context(), req.IDs)
	if err != nil {
		utils.Errorf("delete files: delete failed, ids=%v, error=%v", req.IDs, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete files: success, success=%d, failed=%d", len(success), len(failed))
	response.Success(c, gin.H{
		"success": success,
		"failed":  failed,
	})
}

// GetChannels 获取通道列表
// GET /api/v1/admin/channels
func (h *AdminHandler) GetChannels(c *gin.Context) {
	channels, err := h.channelService.ListChannels(c.Request.Context())
	if err != nil {
		utils.Errorf("get channels: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 返回解密后的配置
	type ChannelWithDecryptedConfig struct {
		model.Channel
		Config map[string]interface{} `json:"config"`
	}
	result := make([]ChannelWithDecryptedConfig, len(channels))
	for i, ch := range channels {
		decryptedConfig, _ := service.DecryptChannelConfig(ch.Config)
		result[i] = ChannelWithDecryptedConfig{
			Channel: ch,
			Config:  decryptedConfig,
		}
	}

	response.Success(c, result)
}

// CreateChannel 创建通道
// POST /api/v1/admin/channels
func (h *AdminHandler) CreateChannel(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Type      string                 `json:"type" binding:"required"`
		Config    map[string]interface{} `json:"config" binding:"required"`
		Quota     model.QuotaConfig      `json:"quota"`
		RateLimit model.RateLimitConfig  `json:"rateLimit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("create channel: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	channel, err := h.channelService.CreateChannel(c.Request.Context(), req.Name, req.Type, req.Config, req.Quota, req.RateLimit)
	if err != nil {
		utils.Errorf("create channel: create failed, name=%s, type=%s, error=%v", req.Name, req.Type, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("create channel: success, channelID=%s, name=%s", channel.ID, req.Name)
	response.Success(c, channel)
}

// UpdateChannel 更新通道
// PUT /api/v1/admin/channels/:id
func (h *AdminHandler) UpdateChannel(c *gin.Context) {
	channelID := c.Param("id")

	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Config    map[string]interface{} `json:"config" binding:"required"`
		Quota     model.QuotaConfig      `json:"quota"`
		RateLimit model.RateLimitConfig  `json:"rateLimit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("update channel: invalid request, channelID=%s, error=%v", channelID, err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.channelService.UpdateChannel(c.Request.Context(), channelID, req.Name, req.Config, req.Quota, req.RateLimit); err != nil {
		utils.Errorf("update channel: update failed, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update channel: success, channelID=%s", channelID)
	response.Success(c, nil)
}

// DeleteChannel 删除通道
// DELETE /api/v1/admin/channels/:id
func (h *AdminHandler) DeleteChannel(c *gin.Context) {
	channelID := c.Param("id")

	if err := h.channelService.DeleteChannel(c.Request.Context(), channelID); err != nil {
		utils.Errorf("delete channel: delete failed, channelID=%s, error=%v", channelID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete channel: success, channelID=%s", channelID)
	response.Success(c, nil)
}

// EnableChannel 启用/禁用通道
// PUT /api/v1/admin/channels/:id/enable
func (h *AdminHandler) EnableChannel(c *gin.Context) {
	channelID := c.Param("id")

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

// TestChannel 测试通道连接
// POST /api/v1/admin/channels/:id/test
func (h *AdminHandler) TestChannel(c *gin.Context) {
	channelID := c.Param("id")

	if err := h.channelService.HealthCheck(c.Request.Context(), channelID); err != nil {
		utils.Errorf("test channel: health check failed, channelID=%s, error=%v", channelID, err)
		response.Success(c, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response.Success(c, gin.H{
		"success": true,
	})
}

// GetSettings 获取系统设置
// GET /api/v1/admin/settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	siteConfig, _ := h.configService.GetSiteConfig()
	uploadConfig, _ := h.configService.GetUploadConfig()

	response.Success(c, gin.H{
		"site":   siteConfig,
		"upload": uploadConfig,
	})
}

// UpdateSettings 更新系统设置
// PUT /api/v1/admin/settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req struct {
		Site   *service.SiteConfig   `json:"site"`
		Upload *service.UploadConfig `json:"upload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("update settings: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if req.Site != nil {
		if err := h.configService.UpdateSiteConfig(req.Site); err != nil {
			utils.Errorf("update settings: update site config failed, error=%v", err)
			response.Error(c, response.ErrInternal, err.Error())
			return
		}
	}

	if req.Upload != nil {
		if err := h.configService.UpdateUploadConfig(req.Upload); err != nil {
			utils.Errorf("update settings: update upload config failed, error=%v", err)
			response.Error(c, response.ErrInternal, err.Error())
			return
		}
	}

	utils.Infof("update settings: success")
	response.Success(c, nil)
}

// GetTokens 获取API Token列表
// GET /api/v1/admin/tokens
func (h *AdminHandler) GetTokens(c *gin.Context) {
	tokens, err := h.tokenService.ListTokens(c.Request.Context())
	if err != nil {
		utils.Errorf("get tokens: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 转换为统一格式，使用Unix时间戳
	result := make([]gin.H, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, gin.H{
			"name":        t.Name,
			"token":       t.Token,
			"permissions": t.Permissions,
			"enabled":     t.Enabled,
			"expiresAt":   t.ExpiresAt.Unix(),
			"createdAt":   t.CreatedAt.Unix(),
			"lastUsedAt":  t.LastUsedAt.Unix(),
		})
	}

	response.Success(c, result)
}

// CreateToken 创建API Token
// POST /api/v1/admin/tokens
func (h *AdminHandler) CreateToken(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
		ExpiresIn   int      `json:"expiresIn"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("create token: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if len(req.Permissions) == 0 {
		req.Permissions = []string{"upload", "read"}
	}

	token, err := h.tokenService.CreateToken(c.Request.Context(), req.Name, req.Permissions, req.ExpiresIn)
	if err != nil {
		utils.Errorf("create token: create failed, name=%s, error=%v", req.Name, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("create token: success, name=%s", req.Name)
	response.Success(c, gin.H{
		"id":          token.Token,
		"token":       token.Token,
		"secret":      token.Secret,
		"name":        token.Name,
		"permissions": token.Permissions,
		"expiresAt":   token.ExpiresAt.Unix(),
		"createdAt":   token.CreatedAt.Unix(),
	})
}

// DeleteToken 删除API Token
// DELETE /api/v1/admin/tokens/:id
func (h *AdminHandler) DeleteToken(c *gin.Context) {
	tokenID := c.Param("id")

	if err := h.tokenService.DeleteToken(c.Request.Context(), tokenID); err != nil {
		utils.Errorf("delete token: delete failed, tokenID=%s, error=%v", tokenID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete token: success, tokenID=%s", tokenID)
	response.Success(c, nil)
}

// EnableToken 启用/禁用API Token
// PUT /api/v1/admin/tokens/:id/enable
func (h *AdminHandler) EnableToken(c *gin.Context) {
	tokenID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("enable token: invalid request, tokenID=%s, error=%v", tokenID, err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.tokenService.EnableToken(c.Request.Context(), tokenID, req.Enabled); err != nil {
		utils.Errorf("enable token: update failed, tokenID=%s, enabled=%v, error=%v", tokenID, req.Enabled, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("enable token: success, tokenID=%s, enabled=%v", tokenID, req.Enabled)
	response.Success(c, nil)
}

// GetStatistics 获取统计数据
// GET /api/v1/admin/statistics
func (h *AdminHandler) GetStatistics(c *gin.Context) {
	var dailyStats []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
		Size  int64  `json:"size"`
	}

	database.DB.Model(&model.File{}).
		Select("date(created_at) as date, count(*) as count, sum(size) as size").
		Where("created_at >= date('now', '-30 days')").
		Group("date(created_at)").
		Order("date DESC").
		Find(&dailyStats)

	var channelStats []struct {
		ChannelID   string `json:"channelId"`
		ChannelName string `json:"channelName"`
		Count       int64  `json:"count"`
		Size        int64  `json:"size"`
	}

	database.DB.Table("files").
		Select("files.channel_id, channels.name as channel_name, count(*) as count, sum(files.size) as size").
		Joins("LEFT JOIN channels ON files.channel_id = channels.id").
		Group("files.channel_id").
		Find(&channelStats)

	response.Success(c, gin.H{
		"daily":   dailyStats,
		"channel": channelStats,
	})
}
