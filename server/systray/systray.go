//go:build gui

package systray

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/getlantern/systray"
	"github.com/imgbed/server/config"
)

var (
	serverAddr     string
	onQuit         func()
	shutdownServer func(context.Context) error
	getActualAddr  func() string
)

func Init(addr string, shutdown func(context.Context) error, actualAddrFunc func() string) {
	serverAddr = addr
	shutdownServer = shutdown
	getActualAddr = actualAddrFunc
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

	systray.AddMenuItem("● 服务运行中", "当前状态")

	systray.AddSeparator()

	mOpenAdmin := systray.AddMenuItem("打开管理后台", "在浏览器中打开管理后台")
	mOpenSite := systray.AddMenuItem("打开网站", "在浏览器中打开网站首页")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mOpenAdmin.ClickedCh:
				openBrowser(getBrowserURL("/admin"))
			case <-mOpenSite.ClickedCh:
				openBrowser(getBrowserURL(""))
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

func getBrowserURL(path string) string {
	var addr string
	if getActualAddr != nil {
		addr = getActualAddr()
	} else {
		addr = serverAddr
	}

	parts := strings.Split(addr, ":")
	host := parts[0]
	port := parts[1]

	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}

	return fmt.Sprintf("http://%s:%s%s", host, port, path)
}
