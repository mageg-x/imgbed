//go:build !gui

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imgbed/server/app"
)

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

func main() {
	// 解析命令行参数
	dataDir := flag.String("d", "", "数据目录路径")
	port := flag.Int("p", 0, "HTTP 端口")
	flag.Parse()

	application, err := app.Init(*dataDir, *port)
	if err != nil {
		fmt.Printf("init failed: %v\n", err)
		os.Exit(1)
	}

	go func() {
		if err := application.Start(); err != nil {
			fmt.Printf("server error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}
