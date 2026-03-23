package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	minPageWidth                = 1460
	minPageHeight               = 712
	minToolbarHeight            = 58
	windowChromeWidthAllowance  = 20
	windowChromeHeightAllowance = 46
	minWindowWidth              = minPageWidth + windowChromeWidthAllowance
	minWindowHeight             = minPageHeight + minToolbarHeight + windowChromeHeightAllowance
)

func resolveWebviewUserDataPath() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && cacheDir != "" {
		return filepath.Join(cacheDir, "CodexSwitch", "webview2")
	}

	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "CodexSwitch", "webview2")
	}

	return filepath.Join(".", "CodexSwitch", "webview2")
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "CodexSwitch",
		Width:     minWindowWidth,
		Height:    minWindowHeight,
		MinWidth:  minWindowWidth,
		MinHeight: minWindowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Windows: &windows.Options{
			// Keep one stable WebView2 profile path for both dev and release builds.
			WebviewUserDataPath: resolveWebviewUserDataPath(),
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
