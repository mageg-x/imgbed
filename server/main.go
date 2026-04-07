package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	_ "github.com/imgbed/server/docs"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/router"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

//go:embed all:embed/admin
var adminFS embed.FS

//go:embed all:embed/site
var siteFS embed.FS

// @title ImgBed API
// @version 1.0
// @description ImgBed 是一个开源免费的图床聚合工具，支持多种存储后端
// @termsOfService https://github.com/imgbed

// @contact.name API Support
// @contact.url https://github.com/imgbed
// @contact.email support@imgbed.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey TokenAuth
// @in header
// @name X-API-Token
// @description API Token for authentication

// main 程序入口函数
// 负责初始化配置、日志、数据库，设置路由并启动HTTP服务器
// 服务器支持优雅关闭，处理SIGINT和SIGTERM信号
func main() {
	// 1. 初始化配置系统
	// 从config.yaml读取配置，设置默认值
	if err := config.Init(); err != nil {
		fmt.Printf("init config failed: %v\n", err)
		os.Exit(1)
	}

	// 获取运行模式用于日志初始化
	mode := config.GetString("app.mode")

	// 2. 初始化日志系统
	// 根据运行模式选择不同的日志输出格式
	if err := utils.InitLogger(mode); err != nil {
		fmt.Printf("init logger failed: %v\n", err)
		os.Exit(1)
	}

	// 验证配置项
	warnings := config.Validate()
	for _, w := range warnings {
		utils.Warnf("config validation: %s", w)
	}

	// 3. 初始化数据库连接
	// 创建数据库文件、执行自动迁移、初始化默认数据
	if err := database.Init(); err != nil {
		utils.Fatalf("init database failed: %v", err)
	}

	// 3.1 从数据库同步配置到 viper
	// 数据库是配置的数据源，viper 用于运行时读取
	syncConfigFromDatabase()

	// 4. 初始化JWT配置
	// 从配置文件读取JWT密钥和过期时间
	jwtSecret := config.GetString("jwt.secret")
	jwtExpire := config.GetInt("jwt.expire")
	if jwtSecret == "" {
		jwtSecret = "imgbed-secret-key"
	}
	if jwtExpire == 0 {
		jwtExpire = 86400
	}
	utils.SetJWTConfig(jwtSecret, jwtExpire)

	// 5. 初始化默认存储通道
	// 如果系统中没有任何通道，则创建一个本地存储通道
	initDefaultChannel()

	// 5.1 启动渠道冷却恢复定时任务
	channelService := service.NewChannelService()
	channelService.StartCooldownRecovery()

	// 6. 设置Gin运行模式
	// 自动检测：localhost/127.0.0.1/0.0.0.0 自动启用 debug 模式，无需配置
	host := config.GetString("app.host")
	isLocalhost := host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0"
	if isLocalhost {
		gin.SetMode(gin.DebugMode)
		utils.Info("running in debug mode (localhost detected)")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 7. 创建Gin引擎并设置路由
	r := router.SetupRouter()

	// 8. 配置静态文件服务
	setupStaticFiles(r)

	// 9. 获取监听地址
	port := config.GetInt("app.port")
	addr := fmt.Sprintf("%s:%d", host, port)

	// 10. 创建HTTP服务器
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second, // 读取超时
		WriteTimeout: 30 * time.Second, // 写入超时
	}

	// 11. 启动HTTP服务器（异步）
	go func() {
		utils.Infof("ImgBed server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Fatalf("listen: %s\n", err)
		}
	}()

	// 12. 等待中断信号
	// 支持优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	utils.Info("Shutting down server...")

	// 13. 优雅关闭服务器
	// 给予5秒时间处理未完成的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		utils.Fatalf("Server forced to shutdown: %v", err)
	}

	utils.Info("Server exited")
}

// setupStaticFiles 配置静态文件服务
// /admin/* -> 管理后台
// /assets/* -> 前端静态资源
// / -> 主站首页
// 参数：
//   - r: Gin引擎实例
func setupStaticFiles(r *gin.Engine) {
	// 管理后台根路径 /admin
	r.GET("/admin", func(c *gin.Context) {
		serveAdminFile(c, "/index.html")
	})

	// 管理后台路由 /admin/*
	r.GET("/admin/*filepath", func(c *gin.Context) {
		serveAdminFile(c, c.Param("filepath"))
	})

	// 前端静态资源 /assets/*
	r.GET("/assets/*filepath", func(c *gin.Context) {
		serveSiteFile(c, "/assets"+c.Param("filepath"))
	})

	// 主站 SPA 路由
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路由不处理，由 router.go 处理
		if strings.HasPrefix(path, "/api/") {
			c.String(404, "API not found")
			return
		}

		// favicon 处理
		if path == "/favicon.svg" {
			serveFile(c, siteFS, "embed/site/", "/favicon.svg")
			return
		}

		// 根路径返回 site index.html
		if path == "/" {
			serveFile(c, siteFS, "embed/site/", "/index.html")
			return
		}

		// 尝试从 site 读取（SPA fallback）
		serveFile(c, siteFS, "embed/site/", path)
	})
}

// serveAdminFile 从 adminFS 提供文件（支持 SPA 路由）
func serveAdminFile(c *gin.Context, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := "embed/admin/" + filePath

	f, err := adminFS.Open(fullPath)
	if err != nil {
		f, err = adminFS.Open("embed/admin/index.html")
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(filePath), data)
}

// serveSiteFile 从 siteFS 提供文件（SPA fallback）
func serveSiteFile(c *gin.Context, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := "embed/site/" + filePath

	f, err := siteFS.Open(fullPath)
	if err != nil {
		// SPA fallback - 尝试返回 index.html
		f, err = siteFS.Open("embed/site/index.html")
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(filePath), data)
}

// serveFile 从 embed.FS 提供文件（通用版本，支持 SPA fallback）
func serveFile(c *gin.Context, fsys embed.FS, prefix, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := prefix + filePath

	f, err := fsys.Open(fullPath)
	if err != nil {
		// SPA fallback - 尝试返回 index.html
		indexPath := prefix + "index.html"
		f, err = fsys.Open(indexPath)
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
		fullPath = indexPath
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(fullPath), data)
}

// getContentType 根据文件扩展名返回 Content-Type
func getContentType(path string) string {
	ext := path[strings.LastIndex(path, "."):]
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// syncConfigFromDatabase 从数据库同步配置到 viper
// 数据库是配置的数据源，此函数在数据库初始化后调用
func syncConfigFromDatabase() {
	var configs []model.Config
	if err := database.DB.Find(&configs).Error; err != nil {
		utils.Errorf("sync config from database: query failed, error=%v", err)
		return
	}

	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}

	config.LoadFromMap(configMap)
	utils.Infof("sync config from database: loaded %d configs", len(configs))
}

// initDefaultChannel 初始化默认存储通道
// 如果系统中没有任何通道，则创建一个本地存储通道
func initDefaultChannel() {
	channelService := service.NewChannelService()

	// 检查是否已存在通道
	channels, err := channelService.ListChannels(context.Background())
	if err != nil {
		utils.Errorf("init default channel: list channels failed, error=%v", err)
		return
	}
	if len(channels) > 0 {
		return
	}

	utils.Info("Creating default local storage channel...")

	// 创建默认本地存储通道
	_, err = channelService.CreateChannel(
		context.Background(),
		"Local Storage",
		"local",
		map[string]interface{}{
			"path": "./data/uploads",
		},
		model.QuotaConfig{
			Enabled: false,
		},
		model.RateLimitConfig{},
	)

	if err != nil {
		utils.Warnf("Failed to create default channel: %v", err)
	}
}
