package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/bridge/httpapi"
)

// web/ is populated from frontend/dist by scripts/sync-web-assets.mjs.
// A checked-in placeholder keeps ordinary Go tooling usable before the first
// frontend build.
//
//go:embed all:web
var embeddedWeb embed.FS

func main() {
	listen := flag.String("listen", "", "Web 管理平台监听地址（留空时读取 configs/server.json）")
	allowRemote := flag.Bool("allow-remote-web", false, "显式允许非 loopback Web 监听；公网部署仍建议使用 TLS 反向代理")
	flag.Parse()

	assets, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取 Web 资源失败:", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app := application.New()
	app.Startup(ctx)
	defer app.Shutdown(context.Background())

	platformConfig, err := app.PlatformConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取 Server 配置失败:", err)
		os.Exit(1)
	}
	addr := strings.TrimSpace(*listen)
	if addr == "" {
		addr = strings.TrimSpace(platformConfig.Server.Listen)
	}
	if addr == "" {
		addr = "127.0.0.1:17890"
	}
	server := httpapi.New(app, httpapi.Options{ListenAddr: addr, Assets: assets, AllowRemote: platformConfig.Server.AllowRemoteWeb || *allowRemote})
	fmt.Printf("AI游戏管理器面板 Web 管理平台: http://%s\n", server.Addr())
	if err := server.ListenAndServe(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Web 服务启动失败:", err)
		os.Exit(1)
	}
}
