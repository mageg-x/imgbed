package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/imgbed/server/utils"
	"github.com/spf13/viper"
)

// 全局 viper 实例和同步控制
var (
	once            sync.Once                      // 确保配置只初始化一次
	v               *viper.Viper                   // viper 配置管理实例
	configCache     = make(map[string]interface{}) // 配置缓存
	cacheMutex      sync.RWMutex                   // 缓存读写锁
	cacheExpires    = make(map[string]time.Time)   // 缓存过期时间
	defaultCacheTTL = 5 * time.Second              // 默认缓存时间
)

// Init 初始化配置系统
// 不读取任何配置文件，所有配置从数据库获取
// 仅设置启动必需的基础配置（数据库路径）
// 返回：
//   - error: 初始化失败时的错误
func Init() error {
	var err error
	once.Do(func() {
		v = viper.New()

		// 从环境变量或硬编码默认值设置数据库路径（这是唯一必需的启动配置）
		if dbPath := os.Getenv("IMGBED_DB_PATH"); dbPath != "" {
			v.Set("database.path", dbPath)
		} else {
			v.Set("database.path", "./data/imgbed.db")
		}

		// 从环境变量读取其他可选配置
		if jwtSecret := os.Getenv("IMGBED_JWT_SECRET"); jwtSecret != "" {
			v.Set("jwt.secret", jwtSecret)
		} else {
			v.Set("jwt.secret", "imgbed-secret-key")
		}

		if appMode := os.Getenv("IMGBED_APP_MODE"); appMode != "" {
			v.Set("app.mode", appMode)
		} else {
			v.Set("app.mode", "debug")
		}

		if appHost := os.Getenv("IMGBED_HOST"); appHost != "" {
			v.Set("app.host", appHost)
		} else {
			v.Set("app.host", "0.0.0.0")
		}

		if appPort := os.Getenv("IMGBED_PORT"); appPort != "" {
			v.Set("app.port", appPort)
		} else {
			v.Set("app.port", 8080)
		}
	})
	return err
}

// Validate 验证必要配置项
// 检查关键配置是否有效，返回警告信息列表
// 返回：
//   - []string: 警告信息列表
func Validate() []string {
	var warnings []string

	jwtSecret := GetString("jwt.secret")
	if jwtSecret == "" || jwtSecret == "imgbed-secret-key" {
		warnings = append(warnings, "jwt.secret is using default value, please set IMGBED_JWT_SECRET environment variable")
	}

	appMode := GetString("app.mode")
	if appMode == "release" {
		if jwtSecret == "imgbed-secret-key" || jwtSecret == "imgbed-secret-key-change-in-production" {
			warnings = append(warnings, "CRITICAL: using default JWT secret in release mode is insecure! Set IMGBED_JWT_SECRET environment variable")
		}
		if len(jwtSecret) < 32 {
			warnings = append(warnings, "JWT secret should be at least 32 characters in production")
		}
	}

	dbPath := GetString("database.path")
	if dbPath == "" {
		warnings = append(warnings, "database.path is empty, using default path")
	}

	maxSize := GetInt("upload.maxSize")
	if maxSize <= 0 || maxSize > 100*1024*1024 {
		warnings = append(warnings, "upload.maxSize should be between 1 and 100MB")
	}

	return warnings
}

// Get 获取任意类型配置
// 参数：
//   - key: 配置键
//
// 返回：
//   - interface{}: 配置值
func Get(key string) interface{} {
	return v.Get(key)
}

// Set 设置配置值
// 用于运行时动态修改配置
// 参数：
//   - key: 配置键
//   - value: 配置值
func Set(key string, value interface{}) {
	v.Set(key, value)
	// 清除缓存
	clearCache(key)
}

// AllSettings 获取所有配置
// 返回当前所有配置项的键值对
// 返回：
//   - map[string]interface{}: 所有配置项
func AllSettings() map[string]interface{} {
	return v.AllSettings()
}

// LoadFromMap 从键值对加载配置到 viper（用于从数据库同步）
// 数据库存储的是字符串，会自动转换为适当的类型
// 参数：
//   - configs: 配置键值对 map
func LoadFromMap(configs map[string]string) {
	for key, value := range configs {
		// 尝试转换字符串值到适当的类型
		if value == "true" {
			v.Set(key, true)
		} else if value == "false" {
			v.Set(key, false)
		} else if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			// JSON 字符串
			v.Set(key, value[1:len(value)-1])
		} else if num, err := strconv.Atoi(value); err == nil {
			v.Set(key, num)
		} else {
			v.Set(key, value)
		}
	}
	// 清除所有缓存
	clearAllCache()
	utils.Debugf("config: loaded %d configs from database", len(configs))
}

// clearAllCache 清除所有配置缓存
func clearAllCache() {
	cacheMutex.Lock()
	configCache = make(map[string]interface{})
	cacheExpires = make(map[string]time.Time)
	cacheMutex.Unlock()
}

// getFromCache 从缓存获取配置值
// 参数：
//   - key: 配置键
//
// 返回：
//   - interface{}: 配置值
//   - bool: 是否命中缓存
func getFromCache(key string) (interface{}, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if val, ok := configCache[key]; ok {
		if time.Now().Before(cacheExpires[key]) {
			return val, true
		}
	}
	return nil, false
}

// setCache 设置配置缓存
// 参数：
//   - key: 配置键
//   - value: 配置值
func setCache(key string, value interface{}) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	configCache[key] = value
	cacheExpires[key] = time.Now().Add(defaultCacheTTL)
}

// clearCache 清除指定配置的缓存
// 参数：
//   - key: 配置键
func clearCache(key string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	delete(configCache, key)
	delete(cacheExpires, key)
}

// GetString 获取字符串类型配置（带缓存）
// 参数：
//   - key: 配置键（如"app.name"）
//
// 返回：
//   - string: 配置值
func GetString(key string) string {
	// 尝试从缓存获取
	if val, ok := getFromCache(key); ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	// 从 viper 获取并缓存
	value := v.GetString(key)
	setCache(key, value)
	return value
}

// GetInt 获取整数类型配置（带缓存）
// 参数：
//   - key: 配置键（如"app.port"）
//
// 返回：
//   - int: 配置值
func GetInt(key string) int {
	if val, ok := getFromCache(key); ok {
		if num, ok := val.(int); ok {
			return num
		}
	}

	value := v.GetInt(key)
	setCache(key, value)
	return value
}

func GetInt64(key string) int64 {
	if val, ok := getFromCache(key); ok {
		if num, ok := val.(int64); ok {
			return num
		}
	}

	value := v.GetInt64(key)
	setCache(key, value)
	return value
}

func GetBool(key string) bool {
	// 尝试从缓存获取
	if val, ok := getFromCache(key); ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}

	// 从 viper 获取并缓存
	value := v.GetBool(key)
	setCache(key, value)
	return value
}

// GetStringSlice 获取字符串数组类型配置（带缓存）
// 参数：
//   - key: 配置键（如"anonymous.allowedTypes"）
//
// 返回：
//   - []string: 配置值
func GetStringSlice(key string) []string {
	// 尝试从缓存获取
	if val, ok := getFromCache(key); ok {
		if slice, ok := val.([]string); ok {
			return slice
		}
	}

	// 从 viper 获取并缓存
	value := v.GetStringSlice(key)
	setCache(key, value)
	return value
}

// CDNConfig CDN 代理配置
type CDNConfig struct {
	Enabled  bool   // 是否启用 CDN 代理
	ProxyUrl string // CDN 代理基础地址（Worker 地址）
	CdnUrl   string // CDN 加速地址
}

// GetCDNConfig 获取 CDN 代理配置
func GetCDNConfig() *CDNConfig {
	return &CDNConfig{
		Enabled:  GetBool("cdn.enabled"),
		ProxyUrl: GetString("cdn.proxyUrl"),
		CdnUrl:   GetString("cdn.cdnUrl"),
	}
}

// IsCDNEnabled 检查 CDN 是否启用
func IsCDNEnabled() bool {
	return GetBool("cdn.enabled")
}

// GetCDNProxyUrl 获取 CDN 代理地址
func GetCDNProxyUrl() string {
	return GetString("cdn.proxyUrl")
}
