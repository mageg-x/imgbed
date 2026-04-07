package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

// ConfigHandler 配置处理器，负责处理配置的读取和更新请求
type ConfigHandler struct {
	configService *service.ConfigService // 配置服务引用
}

// NewConfigHandler 创建ConfigHandler实例
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{
		configService: service.NewConfigService(),
	}
}

// Get 获取单个配置项
// GET /api/v1/config/:key (需要admin权限)
func (h *ConfigHandler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		utils.Warnf("get config: config key is required")
		response.ValidationError(c, "config key is required")
		return
	}

	value, err := h.configService.Get(key)
	if err != nil {
		utils.Errorf("get config: query failed, key=%s, error=%v", key, err)
		response.Error(c, response.ErrNotFound, "config not found")
		return
	}

	response.Success(c, gin.H{"key": key, "value": value})
}

// GetAll 获取所有配置项
// GET /api/v1/config (需要admin权限)
func (h *ConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.configService.GetAll()
	if err != nil {
		utils.Errorf("get all config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, configs)
}

// Set 设置配置项
// PUT /api/v1/config (需要admin权限)
func (h *ConfigHandler) Set(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("set config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.Set(req.Key, req.Value); err != nil {
		utils.Errorf("set config: set failed, key=%s, error=%v", req.Key, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("set config: success, key=%s", req.Key)
	response.Success(c, nil)
}

// GetUploadConfig 获取上传配置
// GET /api/v1/config/upload (需要admin权限)
func (h *ConfigHandler) GetUploadConfig(c *gin.Context) {
	config, err := h.configService.GetUploadConfig()
	if err != nil {
		utils.Errorf("get upload config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateUploadConfig 更新上传配置
// PUT /api/v1/config/upload (需要admin权限)
func (h *ConfigHandler) UpdateUploadConfig(c *gin.Context) {
	var config service.UploadConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update upload config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateUploadConfig(&config); err != nil {
		utils.Errorf("update upload config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update upload config: success")
	response.Success(c, nil)
}

// GetSiteConfig 获取站点配置
// GET /api/v1/config/site (需要admin权限)
func (h *ConfigHandler) GetSiteConfig(c *gin.Context) {
	config, err := h.configService.GetSiteConfig()
	if err != nil {
		utils.Errorf("get site config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateSiteConfig 更新站点配置
// PUT /api/v1/config/site (需要admin权限)
func (h *ConfigHandler) UpdateSiteConfig(c *gin.Context) {
	var config service.SiteConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update site config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateSiteConfig(&config); err != nil {
		utils.Errorf("update site config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update site config: success")
	response.Success(c, nil)
}

// GetAuthConfig 获取认证配置
// GET /api/v1/config/auth (需要admin权限)
func (h *ConfigHandler) GetAuthConfig(c *gin.Context) {
	config, err := h.configService.GetAuthConfig()
	if err != nil {
		utils.Errorf("get auth config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 隐藏密码
	config.AdminPassword = ""

	response.Success(c, config)
}

// UpdateAuthConfig 更新认证配置
// PUT /api/v1/config/auth (需要admin权限)
func (h *ConfigHandler) UpdateAuthConfig(c *gin.Context) {
	var config service.AuthConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update auth config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateAuthConfig(&config); err != nil {
		utils.Errorf("update auth config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update auth config: success")
	response.Success(c, nil)
}

// GetAccessConfig 获取访问控制配置
// GET /api/v1/config/access (需要admin权限)
func (h *ConfigHandler) GetAccessConfig(c *gin.Context) {
	config, err := h.configService.GetAccessConfig()
	if err != nil {
		utils.Errorf("get access config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateAccessConfig 更新访问控制配置
// PUT /api/v1/config/access (需要admin权限)
func (h *ConfigHandler) UpdateAccessConfig(c *gin.Context) {
	var config service.AccessConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update access config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateAccessConfig(&config); err != nil {
		utils.Errorf("update access config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update access config: success")
	response.Success(c, nil)
}

// GetRateLimitConfig 获取速率限制配置
// GET /api/v1/config/rate-limit (需要admin权限)
func (h *ConfigHandler) GetRateLimitConfig(c *gin.Context) {
	config, err := h.configService.GetRateLimitConfig()
	if err != nil {
		utils.Errorf("get rate limit config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateRateLimitConfig 更新速率限制配置
// PUT /api/v1/config/rate-limit (需要admin权限)
func (h *ConfigHandler) UpdateRateLimitConfig(c *gin.Context) {
	var config service.RateLimitConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update rate limit config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateRateLimitConfig(&config); err != nil {
		utils.Errorf("update rate limit config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update rate limit config: success")
	response.Success(c, nil)
}

// GetModerationConfig 获取内容审核配置
// GET /api/v1/config/moderation (需要admin权限)
func (h *ConfigHandler) GetModerationConfig(c *gin.Context) {
	config, err := h.configService.GetModerationConfig()
	if err != nil {
		utils.Errorf("get moderation config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 隐藏API Key
	config.APIKey = ""

	response.Success(c, config)
}

// UpdateModerationConfig 更新内容审核配置
// PUT /api/v1/config/moderation (需要admin权限)
func (h *ConfigHandler) UpdateModerationConfig(c *gin.Context) {
	var config service.ModerationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update moderation config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateModerationConfig(&config); err != nil {
		utils.Errorf("update moderation config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update moderation config: success")
	response.Success(c, nil)
}

// GetScheduleConfig 获取调度策略配置
// GET /api/v1/config/schedule (需要admin权限)
func (h *ConfigHandler) GetScheduleConfig(c *gin.Context) {
	config, err := h.configService.GetScheduleConfig()
	if err != nil {
		utils.Errorf("get schedule config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateScheduleConfig 更新调度策略配置
// PUT /api/v1/config/schedule (需要admin权限)
func (h *ConfigHandler) UpdateScheduleConfig(c *gin.Context) {
	var config service.ScheduleConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update schedule config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateScheduleConfig(&config); err != nil {
		utils.Errorf("update schedule config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update schedule config: success")
	response.Success(c, nil)
}

// GetAppConfig 获取应用配置
// GET /api/v1/config/app (需要admin权限)
func (h *ConfigHandler) GetAppConfig(c *gin.Context) {
	config, err := h.configService.GetAppConfig()
	if err != nil {
		utils.Errorf("get app config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, config)
}

// UpdateAppConfig 更新应用配置
// PUT /api/v1/config/app (需要admin权限)
func (h *ConfigHandler) UpdateAppConfig(c *gin.Context) {
	var config service.AppConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update app config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateAppConfig(&config); err != nil {
		utils.Errorf("update app config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update app config: success")
	response.Success(c, nil)
}

// GetJwtConfig 获取JWT配置
// GET /api/v1/config/jwt (需要admin权限)
func (h *ConfigHandler) GetJwtConfig(c *gin.Context) {
	config, err := h.configService.GetJwtConfig()
	if err != nil {
		utils.Errorf("get jwt config: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 隐藏真实secret
	config.Secret = ""

	response.Success(c, config)
}

// UpdateJwtConfig 更新JWT配置
// PUT /api/v1/config/jwt (需要admin权限)
func (h *ConfigHandler) UpdateJwtConfig(c *gin.Context) {
	var config service.JwtConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.Warnf("update jwt config: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.configService.UpdateJwtConfig(&config); err != nil {
		utils.Errorf("update jwt config: update failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("update jwt config: success")
	response.Success(c, nil)
}
