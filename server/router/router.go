package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/imgbed/server/handler"
	"github.com/imgbed/server/middleware"
	"github.com/imgbed/server/service"
)

func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(middleware.CSRFProtection())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		authHandler := handler.NewAuthHandler()
		fileHandler := handler.NewFileHandler()
		channelHandler := handler.NewChannelHandler()
		statsHandler := handler.NewStatsHandler()
		tokenHandler := handler.NewTokenHandler()
		configHandler := handler.NewConfigHandler()
		adminHandler := handler.NewAdminHandler()
		backupService := service.NewBackupService()
		backupHandler := handler.NewBackupHandler(backupService)

		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/admin/login", authHandler.AdminLogin)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/check", middleware.AuthRequired(), authHandler.CheckAuth)
			auth.POST("/refresh", middleware.AuthRequired(), authHandler.RefreshToken)
			auth.POST("/password", middleware.AdminRequired(), authHandler.SetPassword)
		}

		file := api.Group("/file")
		{
			file.GET("/check/:checksum", fileHandler.CheckChecksum)
			file.GET("/:id", fileHandler.GetFile)
			file.GET("/:id/info", fileHandler.GetInfo)
			file.GET("/:id/download", fileHandler.Download)
			file.GET("/:id/proxy", fileHandler.Proxy)
		}

		upload := api.Group("/upload")
		upload.Use(middleware.RateLimitFromConfig())
		upload.Use(middleware.AuthRequired())
		upload.Use(middleware.TokenPermissionRequired("upload"))
		{
			upload.POST("", fileHandler.Upload)
			upload.POST("/multiple", middleware.TokenPermissionRequired("upload:multiple"), fileHandler.UploadMultiple)
		}

		token := api.Group("/tokens")
		{
			token.GET("", middleware.AuthRequired(), tokenHandler.ListTokens)
			token.POST("", middleware.AuthRequired(), tokenHandler.CreateToken)
			token.DELETE("/:token", middleware.AuthRequired(), tokenHandler.DeleteToken)
			token.PUT("/:token/enable", middleware.AuthRequired(), tokenHandler.EnableToken)
		}

		files := api.Group("/files")
		{
			files.GET("", middleware.AuthRequired(), fileHandler.List)
			files.GET("/ids", middleware.AuthRequired(), fileHandler.ListIds)
			files.DELETE("", middleware.AuthRequired(), middleware.TokenPermissionRequired("delete"), fileHandler.DeleteMultiple)
			files.POST("/cleanup/preview", middleware.AuthRequired(), fileHandler.CleanupPreview)
			files.POST("/cleanup", middleware.AuthRequired(), fileHandler.Cleanup)
		}

		fileManage := api.Group("/file")
		fileManage.Use(middleware.AuthRequired())
		fileManage.Use(middleware.TokenPermissionRequired("delete"))
		{
			fileManage.DELETE("/:id", fileHandler.Delete)
		}

		channel := api.Group("/channel")
		channel.Use(middleware.AuthRequired())
		{
			channel.GET("", channelHandler.List)
			channel.GET("/:id", channelHandler.Get)
			channel.GET("/:id/status", channelHandler.GetStatus)
			channel.GET("/:id/stats", channelHandler.GetChannelStats)
			channel.POST("", middleware.AdminRequired(), channelHandler.Create)
			channel.PUT("/:id", middleware.AdminRequired(), channelHandler.Update)
			channel.DELETE("/:id", middleware.AdminRequired(), channelHandler.Delete)
			channel.PUT("/:id/enable", middleware.AdminRequired(), channelHandler.Enable)
			channel.PUT("/:id/weight", middleware.AdminRequired(), channelHandler.SetWeight)
			channel.GET("/:id/health", middleware.AdminRequired(), channelHandler.HealthCheck)
			channel.POST("/:id/test", middleware.AdminRequired(), channelHandler.TestChannel)
		}

		channels := api.Group("/channels")
		{
			channels.GET("/status", middleware.AuthRequired(), channelHandler.GetAllStatus)
			channels.POST("/health-check", middleware.AdminRequired(), channelHandler.HealthCheckAll)
		}

		api.GET("/config/upload", configHandler.GetUploadConfig)

		config := api.Group("/config")
		config.Use(middleware.AdminRequired())
		{
			config.GET("", configHandler.GetAll)
			config.GET("/:key", configHandler.Get)
			config.PUT("", configHandler.Set)
			config.PUT("/upload", configHandler.UpdateUploadConfig)
			config.GET("/site", configHandler.GetSiteConfig)
			config.PUT("/site", configHandler.UpdateSiteConfig)
			config.GET("/auth", configHandler.GetAuthConfig)
			config.PUT("/auth", configHandler.UpdateAuthConfig)
			config.GET("/rate-limit", configHandler.GetRateLimitConfig)
			config.PUT("/rate-limit", configHandler.UpdateRateLimitConfig)
			config.GET("/schedule", configHandler.GetScheduleConfig)
			config.PUT("/schedule", configHandler.UpdateScheduleConfig)
			config.GET("/app", configHandler.GetAppConfig)
			config.PUT("/app", configHandler.UpdateAppConfig)
			config.GET("/jwt", configHandler.GetJwtConfig)
			config.PUT("/jwt", configHandler.UpdateJwtConfig)
			config.GET("/cdn", configHandler.GetCDNConfig)
			config.PUT("/cdn", configHandler.UpdateCDNConfig)
			config.GET("/backup", configHandler.GetBackupConfig)
			config.PUT("/backup", configHandler.UpdateBackupConfig)
		}

		stats := api.Group("/stats")
		{
			stats.GET("/overview", statsHandler.Overview)
			stats.GET("/channels", statsHandler.Channels)
			stats.GET("/trend", statsHandler.Trend)
			stats.GET("/weekly", statsHandler.Weekly)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/dashboard", adminHandler.Dashboard)
			admin.GET("/statistics", adminHandler.GetStatistics)
			admin.GET("/files", adminHandler.GetFiles)
			admin.DELETE("/files/:id", adminHandler.DeleteFile)
			admin.DELETE("/files", adminHandler.DeleteFiles)
			admin.GET("/channels", adminHandler.GetChannels)
			admin.POST("/channels", adminHandler.CreateChannel)
			admin.PUT("/channels/:id", adminHandler.UpdateChannel)
			admin.DELETE("/channels/:id", adminHandler.DeleteChannel)
			admin.PUT("/channels/:id/enable", adminHandler.EnableChannel)
			admin.POST("/channels/:id/test", adminHandler.TestChannel)
			admin.GET("/settings", adminHandler.GetSettings)
			admin.PUT("/settings", adminHandler.UpdateSettings)
			admin.GET("/tokens", adminHandler.GetTokens)
			admin.POST("/tokens", adminHandler.CreateToken)
			admin.DELETE("/tokens/:id", adminHandler.DeleteToken)
			admin.PUT("/tokens/:id/enable", adminHandler.EnableToken)
			// 备份管理
			admin.GET("/backup/list", backupHandler.ListBackups)
			admin.POST("/backup/create", backupHandler.CreateBackup)
			admin.DELETE("/backup", backupHandler.DeleteBackup)
			admin.POST("/backup/restore", backupHandler.RestoreBackup)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "not found",
		})
	})

	return r
}
