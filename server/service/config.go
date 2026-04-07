package service

import (
	"strconv"
	"strings"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/utils"
)

// ConfigService 配置服务，负责系统配置的读取和修改
type ConfigService struct{}

// NewConfigService 创建ConfigService实例
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// Get 获取指定配置项的值
// 参数：
//   - key: 配置项的键
//
// 返回：
//   - string: 配置值
//   - error: 获取过程中的错误
func (s *ConfigService) Get(key string) (string, error) {
	var cfg model.Config
	if err := database.DB.Where("key = ?", key).First(&cfg).Error; err != nil {
		utils.Errorf("get config: query failed, key=%s, error=%v", key, err)
		return "", err
	}
	return cfg.Value, nil
}

// GetInt 获取指定配置项的值作为整数
// 参数：
//   - key: 配置项的键
//
// 返回：
//   - int: 配置值（整数）
//   - error: 获取过程中的错误
func (s *ConfigService) GetInt(key string) (int, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	// 解析整数值
	result := 0
	for _, c := range value {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result, nil
}

// GetInt64 获取指定配置项的值作为64位整数
// 参数：
//   - key: 配置项的键
//
// 返回：
//   - int64: 配置值（64位整数）
//   - error: 获取过程中的错误
func (s *ConfigService) GetInt64(key string) (int64, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	// 解析64位整数值
	var result int64
	for _, c := range value {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		}
	}
	return result, nil
}

// GetBool 获取指定配置项的值作为布尔值
// 参数：
//   - key: 配置项的键
//
// 返回：
//   - bool: 配置值（布尔值）
//   - error: 获取过程中的错误
func (s *ConfigService) GetBool(key string) (bool, error) {
	value, err := s.Get(key)
	if err != nil {
		return false, err
	}
	return value == "true" || value == "1", nil
}

// Set 设置配置项的值（如果不存在则创建，存在则更新）
// 参数：
//   - key: 配置项的键
//   - value: 配置项的值
//
// 返回：
//   - error: 设置过程中的错误
func (s *ConfigService) Set(key, value string) error {
	var cfg model.Config
	result := database.DB.Where("key = ?", key).First(&cfg)

	if result.Error != nil {
		// 配置项不存在，创建新的
		cfg = model.Config{Key: key, Value: value}
		if err := database.DB.Create(&cfg).Error; err != nil {
			utils.Errorf("set config: create failed, key=%s, error=%v", key, err)
			return err
		}
	} else {
		// 配置项已存在，更新值
		cfg.Value = value
		if err := database.DB.Save(&cfg).Error; err != nil {
			utils.Errorf("set config: update failed, key=%s, error=%v", key, err)
			return err
		}
	}

	// 同步更新 viper 运行时配置
	config.Set(key, value)
	return nil
}

// GetAll 获取所有配置项
// 返回：
//   - map[string]string: 所有配置项的键值对
//   - error: 获取过程中的错误
func (s *ConfigService) GetAll() (map[string]string, error) {
	var configs []model.Config
	if err := database.DB.Find(&configs).Error; err != nil {
		utils.Errorf("get all config: query failed, error=%v", err)
		return nil, err
	}

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	return result, nil
}

// GetByPrefix 获取所有以指定前缀开头的配置项
// 参数：
//   - prefix: 配置项键的前缀
//
// 返回：
//   - map[string]string: 匹配的配置项键值对
//   - error: 获取过程中的错误
func (s *ConfigService) GetByPrefix(prefix string) (map[string]string, error) {
	var configs []model.Config
	if err := database.DB.Where("key LIKE ?", prefix+"%").Find(&configs).Error; err != nil {
		utils.Errorf("get config by prefix: query failed, prefix=%s, error=%v", prefix, err)
		return nil, err
	}

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	return result, nil
}

// GetUploadConfig 获取上传配置
// 返回：
//   - *UploadConfig: 上传配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetUploadConfig() (*UploadConfig, error) {
	configs, err := s.GetByPrefix("upload.")
	if err != nil {
		utils.Errorf("get upload config: get by prefix failed, error=%v", err)
		return nil, err
	}

	// 解析最大文件大小，默认20MB
	maxSize, _ := strconv.ParseInt(configs["upload.max_size"], 10, 64)
	if maxSize == 0 {
		maxSize = 20 * 1024 * 1024
	}

	// 解析分块大小，默认5MB
	chunkSize, _ := strconv.ParseInt(configs["upload.chunk_size"], 10, 64)
	if chunkSize == 0 {
		chunkSize = 5 * 1024 * 1024
	}

	// 解析重试次数，默认3次
	retryCount, _ := strconv.Atoi(configs["upload.retry_count"])
	if retryCount == 0 {
		retryCount = 3
	}

	// 解析压缩质量，默认80
	quality, _ := strconv.Atoi(configs["compression.quality"])
	if quality == 0 {
		quality = 80
	}

	// 解析最大宽高
	maxWidth, _ := strconv.Atoi(configs["compression.max_width"])
	if maxWidth == 0 {
		maxWidth = 1920
	}
	maxHeight, _ := strconv.Atoi(configs["compression.max_height"])
	if maxHeight == 0 {
		maxHeight = 1080
	}

	return &UploadConfig{
		MaxSize:        maxSize,
		ChunkSize:      chunkSize,
		DefaultChannel: configs["upload.default_channel"],
		AllowedTypes:   strings.Split(configs["upload.allowed_types"], ","),
		AutoRetry:      configs["upload.auto_retry"] == "true",
		RetryCount:     retryCount,
		Compression: CompressionConfig{
			Enabled:   configs["compression.enabled"] != "false",
			Quality:   quality,
			Format:    configs["compression.format"],
			MaxWidth:  maxWidth,
			MaxHeight: maxHeight,
		},
	}, nil
}

// GetSiteConfig 获取站点配置
// 返回：
//   - *SiteConfig: 站点配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetSiteConfig() (*SiteConfig, error) {
	configs, err := s.GetByPrefix("site.")
	if err != nil {
		utils.Errorf("get site config: get by prefix failed, error=%v", err)
		return nil, err
	}

	return &SiteConfig{
		Name:       configs["site.name"],
		Logo:       configs["site.logo"],
		Background: configs["site.background"],
		FooterText: configs["site.footer_text"],
	}, nil
}

// GetAuthConfig 获取认证配置
// 返回：
//   - *AuthConfig: 认证配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetAuthConfig() (*AuthConfig, error) {
	configs, err := s.GetByPrefix("auth.")
	if err != nil {
		utils.Errorf("get auth config: get by prefix failed, error=%v", err)
		return nil, err
	}

	// 解析会话超时时间，默认86400秒（24小时）
	timeout, _ := strconv.Atoi(configs["auth.session_timeout"])
	if timeout == 0 {
		timeout = 86400
	}

	return &AuthConfig{
		UserPassword:   configs["auth.user_password"],
		AdminUsername:  configs["auth.admin_username"],
		AdminPassword:  configs["auth.admin_password"],
		SessionTimeout: timeout,
	}, nil
}

// GetAccessConfig 获取访问控制配置
// 返回：
//   - *AccessConfig: 访问控制配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetAccessConfig() (*AccessConfig, error) {
	configs, err := s.GetByPrefix("access.")
	if err != nil {
		utils.Errorf("get access config: get by prefix failed, error=%v", err)
		return nil, err
	}

	// 解析允许的域名列表
	domains := strings.Split(configs["access.allowed_domains"], ",")
	if configs["access.allowed_domains"] == "" {
		domains = []string{}
	}

	return &AccessConfig{
		AllowedDomains: domains,
		WhitelistMode:  configs["access.whitelist_mode"] == "true",
	}, nil
}

// GetRateLimitConfig 获取速率限制配置
// 返回：
//   - *RateLimitConfig: 速率限制配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetRateLimitConfig() (*RateLimitConfig, error) {
	configs, err := s.GetByPrefix("rate_limit.")
	if err != nil {
		utils.Errorf("get rate limit config: get by prefix failed, error=%v", err)
		return nil, err
	}

	// 解析每分钟请求数限制，默认60
	requestsPerMinute, _ := strconv.Atoi(configs["rate_limit.requests_per_minute"])
	if requestsPerMinute == 0 {
		requestsPerMinute = 60
	}

	// 解析每小时上传数限制，默认100
	uploadsPerHour, _ := strconv.Atoi(configs["rate_limit.uploads_per_hour"])
	if uploadsPerHour == 0 {
		uploadsPerHour = 100
	}

	return &RateLimitConfig{
		Enabled:           configs["rate_limit.enabled"] == "true",
		RequestsPerMinute: requestsPerMinute,
		UploadsPerHour:    uploadsPerHour,
	}, nil
}

// GetModerationConfig 获取内容审核配置
// 返回：
//   - *ModerationConfig: 内容审核配置对象
//   - error: 获取过程中的错误
func (s *ConfigService) GetModerationConfig() (*ModerationConfig, error) {
	configs, err := s.GetByPrefix("moderation.")
	if err != nil {
		utils.Errorf("get moderation config: get by prefix failed, error=%v", err)
		return nil, err
	}

	return &ModerationConfig{
		Enabled:  configs["moderation.enabled"] == "true",
		Provider: configs["moderation.provider"],
		APIKey:   configs["moderation.api_key"],
	}, nil
}

// UpdateUploadConfig 更新上传配置
// 参数：
//   - cfg: 上传配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateUploadConfig(cfg *UploadConfig) error {
	updates := map[string]string{
		"upload.max_size":        strconv.FormatInt(cfg.MaxSize, 10),
		"upload.chunk_size":      strconv.FormatInt(cfg.ChunkSize, 10),
		"upload.default_channel": cfg.DefaultChannel,
		"upload.allowed_types":   strings.Join(cfg.AllowedTypes, ","),
		"upload.auto_retry":      strconv.FormatBool(cfg.AutoRetry),
		"upload.retry_count":     strconv.Itoa(cfg.RetryCount),
		"compression.enabled":    strconv.FormatBool(cfg.Compression.Enabled),
		"compression.quality":    strconv.Itoa(cfg.Compression.Quality),
		"compression.format":     cfg.Compression.Format,
		"compression.max_width":  strconv.Itoa(cfg.Compression.MaxWidth),
		"compression.max_height": strconv.Itoa(cfg.Compression.MaxHeight),
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update upload config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UpdateSiteConfig 更新站点配置
// 参数：
//   - cfg: 站点配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateSiteConfig(cfg *SiteConfig) error {
	updates := map[string]string{
		"site.name":        cfg.Name,
		"site.logo":        cfg.Logo,
		"site.background":  cfg.Background,
		"site.footer_text": cfg.FooterText,
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update site config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UpdateAuthConfig 更新认证配置
// 参数：
//   - cfg: 认证配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateAuthConfig(cfg *AuthConfig) error {
	updates := map[string]string{
		"auth.user_password":   cfg.UserPassword,
		"auth.admin_username":  cfg.AdminUsername,
		"auth.session_timeout": strconv.Itoa(cfg.SessionTimeout),
	}

	// 只有当AdminPassword不为空时才更新
	if cfg.AdminPassword != "" {
		updates["auth.admin_password"] = cfg.AdminPassword
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update auth config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UpdateAccessConfig 更新访问控制配置
// 参数：
//   - cfg: 访问控制配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateAccessConfig(cfg *AccessConfig) error {
	updates := map[string]string{
		"access.allowed_domains": strings.Join(cfg.AllowedDomains, ","),
		"access.whitelist_mode":  strconv.FormatBool(cfg.WhitelistMode),
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update access config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UpdateRateLimitConfig 更新速率限制配置
// 参数：
//   - cfg: 速率限制配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateRateLimitConfig(cfg *RateLimitConfig) error {
	updates := map[string]string{
		"rate_limit.enabled":             strconv.FormatBool(cfg.Enabled),
		"rate_limit.requests_per_minute": strconv.Itoa(cfg.RequestsPerMinute),
		"rate_limit.uploads_per_hour":    strconv.Itoa(cfg.UploadsPerHour),
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update rate limit config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UpdateModerationConfig 更新内容审核配置
// 参数：
//   - cfg: 内容审核配置对象
//
// 返回：
//   - error: 更新过程中的错误
func (s *ConfigService) UpdateModerationConfig(cfg *ModerationConfig) error {
	updates := map[string]string{
		"moderation.enabled":  strconv.FormatBool(cfg.Enabled),
		"moderation.provider": cfg.Provider,
		"moderation.api_key":  cfg.APIKey,
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update moderation config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// UploadConfig 上传配置结构
type UploadConfig struct {
	MaxSize        int64             // 最大文件大小（字节）
	ChunkSize      int64             // 分块上传大小（字节）
	DefaultChannel string            // 默认存储通道
	AllowedTypes   []string          // 允许的文件类型
	AutoRetry      bool              // 是否自动重试
	RetryCount     int               // 重试次数
	Compression    CompressionConfig // 图片压缩配置
}

// CompressionConfig 图片压缩配置
type CompressionConfig struct {
	Enabled   bool   // 是否启用压缩
	Quality   int    // 压缩质量 (1-100)
	Format    string // 目标格式 (original/webp/jpeg/png)
	MaxWidth  int    // 最大宽度
	MaxHeight int    // 最大高度
}

// SiteConfig 站点配置结构
type SiteConfig struct {
	Name       string // 站点名称
	Logo       string // 站点Logo
	Background string // 背景图片
	FooterText string // 页脚文本
}

// AuthConfig 认证配置结构
type AuthConfig struct {
	UserPassword   string // 用户密码
	AdminUsername  string // 管理员用户名
	AdminPassword  string // 管理员密码
	SessionTimeout int    // 会话超时时间（秒）
}

// AccessConfig 访问控制配置结构
type AccessConfig struct {
	AllowedDomains []string // 允许的域名列表
	WhitelistMode  bool     // 白名单模式
}

// RateLimitConfig 速率限制配置结构
type RateLimitConfig struct {
	Enabled           bool // 是否启用
	RequestsPerMinute int  // 每分钟请求数限制
	UploadsPerHour    int  // 每小时上传数限制
}

// ModerationConfig 内容审核配置结构
type ModerationConfig struct {
	Enabled  bool   // 是否启用审核
	Provider string // 审核服务提供商
	APIKey   string // API密钥
}

// ScheduleConfig 调度策略配置结构
type ScheduleConfig struct {
	Strategy string // 调度策略: priority, weight, round-robin, random
}

// AppConfig 应用配置结构
type AppConfig struct {
	Host string // 监听地址
	Port int    // 监听端口
	Mode string // 运行模式
}

// JwtConfig JWT配置结构
type JwtConfig struct {
	Secret string // JWT密钥
	Expire int    // 过期时间(秒)
}

// GetAppConfig 获取应用配置
func (s *ConfigService) GetAppConfig() (*AppConfig, error) {
	configs, err := s.GetByPrefix("app.")
	if err != nil {
		utils.Errorf("get app config: get by prefix failed, error=%v", err)
		return nil, err
	}

	port, _ := strconv.Atoi(configs["app.port"])
	if port == 0 {
		port = 8080
	}

	return &AppConfig{
		Host: configs["app.host"],
		Port: port,
		Mode: configs["app.mode"],
	}, nil
}

// UpdateAppConfig 更新应用配置
func (s *ConfigService) UpdateAppConfig(cfg *AppConfig) error {
	updates := map[string]string{
		"app.host": cfg.Host,
		"app.port": strconv.Itoa(cfg.Port),
		"app.mode": cfg.Mode,
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update app config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// GetJwtConfig 获取JWT配置
func (s *ConfigService) GetJwtConfig() (*JwtConfig, error) {
	configs, err := s.GetByPrefix("jwt.")
	if err != nil {
		utils.Errorf("get jwt config: get by prefix failed, error=%v", err)
		return nil, err
	}

	expire, _ := strconv.Atoi(configs["jwt.expire"])
	if expire == 0 {
		expire = 86400
	}

	return &JwtConfig{
		Secret: configs["jwt.secret"],
		Expire: expire,
	}, nil
}

// UpdateJwtConfig 更新JWT配置
func (s *ConfigService) UpdateJwtConfig(cfg *JwtConfig) error {
	updates := map[string]string{
		"jwt.expire": strconv.Itoa(cfg.Expire),
	}

	// 只有当Secret不为空时才更新
	if cfg.Secret != "" {
		updates["jwt.secret"] = cfg.Secret
	}

	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			utils.Errorf("update jwt config: set failed, key=%s, error=%v", key, err)
			return err
		}
	}
	return nil
}

// GetScheduleConfig 获取调度策略配置
func (s *ConfigService) GetScheduleConfig() (*ScheduleConfig, error) {
	configs, err := s.GetByPrefix("schedule.")
	if err != nil {
		utils.Errorf("get schedule config: get by prefix failed, error=%v", err)
		return nil, err
	}

	strategy := configs["schedule.strategy"]
	if strategy == "" {
		strategy = "priority"
	}

	return &ScheduleConfig{
		Strategy: strategy,
	}, nil
}

// UpdateScheduleConfig 更新调度策略配置
func (s *ConfigService) UpdateScheduleConfig(cfg *ScheduleConfig) error {
	if err := s.Set("schedule.strategy", cfg.Strategy); err != nil {
		utils.Errorf("update schedule config: set failed, error=%v", err)
		return err
	}
	return nil
}
