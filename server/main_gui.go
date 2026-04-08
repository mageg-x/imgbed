//go:build gui

package main

import (
	"fmt"
	"os"

	"github.com/imgbed/server/app"
	"github.com/imgbed/server/systray"
)

func main() {
	application, err := app.Init()
	if err != nil {
		fmt.Printf("init failed: %v\n", err)
		os.Exit(1)
	}

	systray.Init(application.Addr, application.Shutdown)

	systray.Run(func() {
		systray.Setup()
		go func() {
			if err := application.Start(); err != nil {
				fmt.Printf("server error: %v\n", err)
			}
		}()
	})
}
