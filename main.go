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
		Width:     1280,
		Height:    1024,
		MinWidth:  1200,
		MinHeight: 864,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Windows: &windows.Options{
			// Keep one stable WebView2 profile path for both dev and release builds.
			WebviewUserDataPath: resolveWebviewUserDataPath(),
		},
		OnStartup: app.startup,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
