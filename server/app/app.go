package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/static"
	"github.com/imgbed/server/utils"
)

type App struct {
	Server   *http.Server
	Addr     string
	Shutdown func(context.Context) error
}

func Init(dataDir string, port int) (*App, error) {
	if err := config.Init(dataDir, port); err != nil {
		return nil, fmt.Errorf("init config failed: %w", err)
	}

	mode := config.GetString("app.mode")
	if err := utils.InitLogger(mode); err != nil {
		return nil, fmt.Errorf("init logger failed: %w", err)
	}

	warnings := config.Validate()
	for _, w := range warnings {
		utils.Warnf("config validation: %s", w)
	}

	if err := database.Init(); err != nil {
		return nil, fmt.Errorf("init database failed: %w", err)
	}

	syncConfigFromDatabase()

	jwtSecret := config.GetString("jwt.secret")
	jwtExpire := config.GetInt("jwt.expire")
	if jwtSecret == "" {
		jwtSecret = "imgbed-secret-key"
	}
	if jwtExpire == 0 {
		jwtExpire = 86400
	}
	utils.SetJWTConfig(jwtSecret, jwtExpire)

	initDefaultChannel()

	channelService := service.NewChannelService()
	channelService.StartCooldownRecovery()

	// 启动自动备份任务
	backupService := service.NewBackupService()
	go startAutoBackup(backupService)

	host := config.GetString("app.host")
	appPort := config.GetInt("app.port")
	addr := fmt.Sprintf("%s:%d", host, appPort)

	r := static.Setup()

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &App{
		Server: srv,
		Addr:   addr,
		Shutdown: func(ctx context.Context) error {
			utils.Info("Shutting down server...")
			return srv.Shutdown(ctx)
		},
	}, nil
}

func (a *App) Start() error {
	ports := []int{8380, 8381, 8382, 8383, 8384}
	for i, port := range ports {
		addr := fmt.Sprintf("%s:%d", config.GetString("app.host"), port)
		// 先检查端口是否可用
		if ln, err := net.Listen("tcp", addr); err != nil {
			if i < len(ports)-1 {
				utils.Warnf("端口 %d 被占用，自动尝试端口 %d", port, ports[i+1])
				continue
			}
			return fmt.Errorf("所有端口都无法绑定: %v", err)
		} else {
			ln.Close() // 只是检查，关闭后立即尝试监听
		}
		a.Server.Addr = addr
		a.Addr = addr
		utils.Infof("ImgBed server starting on %s", addr)
		if err := a.Server.ListenAndServe(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("所有端口都无法绑定")
}

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

func initDefaultChannel() {
	channelService := service.NewChannelService()

	channels, err := channelService.ListChannels(context.Background())
	if err != nil {
		utils.Errorf("init default channel: list channels failed, error=%v", err)
		return
	}
	if len(channels) > 0 {
		return
	}

	utils.Info("Creating default local storage channel...")

	// 默认上传路径：在数据目录下创建 uploads 子目录
	defaultPath := "./data/uploads"
	if dataDir := config.GetString("app.dataDir"); dataDir != "" {
		defaultPath = dataDir + "/uploads"
	}

	_, err = channelService.CreateChannel(
		context.Background(),
		"Local Storage",
		"local",
		map[string]interface{}{
			"path": defaultPath,
		},
		100,
		model.QuotaConfig{
			Enabled: false,
		},
		model.RateLimitConfig{},
	)

	if err != nil {
		utils.Warnf("Failed to create default channel: %v", err)
	}
}

// 启动自动备份任务
func startAutoBackup(backupService *service.BackupService) {
	for {
		// 读取备份配置
		enabled := config.GetBool("backup.enabled")
		interval := config.GetInt("backup.interval")

		if interval <= 0 {
			interval = 24
		}

		if enabled {
			backupService.AutoBackup()
			utils.Infof("auto backup: completed, next in %d hours", interval)
		}

		// 先睡再执行（启动时立即备份一次）
		time.Sleep(time.Duration(interval) * time.Hour)
	}
}
