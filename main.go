package main

import (
	"embed"
	"fmt"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	wailsbridge "github.com/yubboo/AI-Game-Manager-Panel/internal/bridge/wails"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New()
	desktop := wailsbridge.New(app)

	err := wails.Run(&options.App{
		Title:            application.Name,
		Width:            1280,
		Height:           840,
		MinWidth:         960,
		MinHeight:        640,
		Frameless:        false,
		StartHidden:      false,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 20, G: 18, B: 16, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			desktop,
		},
	})
	if err != nil {
		fmt.Println("AI Game Manager Panel failed to start:", err)
	}
}
