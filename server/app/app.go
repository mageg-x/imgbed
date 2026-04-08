package app

import (
	"context"
	"fmt"
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

func Init() (*App, error) {
	if err := config.Init(); err != nil {
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

	host := config.GetString("app.host")
	port := config.GetInt("app.port")
	addr := fmt.Sprintf("%s:%d", host, port)

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
	utils.Infof("ImgBed server starting on %s", a.Addr)
	return a.Server.ListenAndServe()
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

	_, err = channelService.CreateChannel(
		context.Background(),
		"Local Storage",
		"local",
		map[string]interface{}{
			"path": "./data/uploads",
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
