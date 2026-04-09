//go:build gui

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/imgbed/server/app"
	"github.com/imgbed/server/systray"
)

func main() {
	dataDir := flag.String("d", "", "数据目录路径")
	port := flag.Int("p", 0, "HTTP 端口")
	flag.Parse()

	application, err := app.Init(*dataDir, *port)
	if err != nil {
		fmt.Printf("init failed: %v\n", err)
		os.Exit(1)
	}

	systray.Init(application.Addr, application.Shutdown, func() string {
		return application.Addr
	})

	systray.Run(func() {
		systray.Setup()
		go func() {
			if err := application.Start(); err != nil {
				fmt.Printf("server error: %v\n", err)
			}
		}()
	})
}
