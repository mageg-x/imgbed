//go:build gui

package systray

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/imgbed/server/config"
)

var (
	serverAddr    string
	onQuit        func()
	shutdownServer func(context.Context) error
)

func Init(addr string, shutdown func(context.Context) error) {
	serverAddr = addr
	shutdownServer = shutdown
}

func Run(onReady func()) {
	systray.Run(onReady, onExit)
}

func onExit() {
	if onQuit != nil {
		onQuit()
	}
}

func Setup() {
	systray.SetIcon(getIcon())
	systray.SetTitle("ImgBed")
	systray.SetTooltip("ImgBed 图床服务运行中")

	mStatus := systray.AddMenuItem("服务运行中", "当前状态")
	mStatus.Disable()

	systray.AddSeparator()

	mOpenAdmin := systray.AddMenuItem("打开管理后台", "在浏览器中打开管理后台")
	mOpenSite := systray.AddMenuItem("打开网站", "在浏览器中打开网站首页")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mOpenAdmin.ClickedCh:
				openBrowser(fmt.Sprintf("http://%s/admin", serverAddr))
			case <-mOpenSite.ClickedCh:
				openBrowser(fmt.Sprintf("http://%s", serverAddr))
			case <-mQuit.ClickedCh:
				if shutdownServer != nil {
					ctx := context.Background()
					shutdownServer(ctx)
				}
				systray.Quit()
				return
			}
		}
	}()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}

func SetOnQuit(f func()) {
	onQuit = f
}

func GetServerAddr() string {
	host := config.GetString("app.host")
	port := config.GetInt("app.port")
	return fmt.Sprintf("%s:%d", host, port)
}
