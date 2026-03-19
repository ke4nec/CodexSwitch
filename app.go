package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"codexswitch/internal/codexswitch"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	service *codexswitch.Service
}

func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := codexswitch.NewService(codexswitch.ServiceOptions{
		Logger: logger,
	})
	if err != nil {
		panic(err)
	}

	return &App{service: service}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppState() (codexswitch.AppState, error) {
	return a.service.GetAppState()
}

func (a *App) ImportCurrentProfile() (codexswitch.AppState, error) {
	return a.service.ImportCurrentProfile()
}

func (a *App) ImportOfficialProfileFile() (codexswitch.AppState, error) {
	if a.ctx == nil {
		return codexswitch.AppState{}, errors.New("Wails runtime 未就绪")
	}

	selectedFile, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择官方账号文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return codexswitch.AppState{}, err
	}
	if strings.TrimSpace(selectedFile) == "" {
		return codexswitch.AppState{}, errors.New("已取消文件选择")
	}

	return a.service.ImportOfficialProfileFile(selectedFile)
}

func (a *App) CreateApiProfile(input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return a.service.CreateAPIProfile(input)
}

func (a *App) UpdateApiProfile(id string, input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return a.service.UpdateAPIProfile(id, input)
}

func (a *App) GetApiProfileInput(id string) (codexswitch.APIProfileInput, error) {
	return a.service.GetAPIProfileInput(id)
}

func (a *App) SwitchProfile(id string) (codexswitch.AppState, error) {
	return a.service.SwitchProfile(id)
}

func (a *App) DeleteProfile(id string) (codexswitch.AppState, error) {
	return a.service.DeleteProfile(id)
}

func (a *App) RefreshRateLimits(ids []string) (codexswitch.AppState, error) {
	return a.service.RefreshRateLimits(ids)
}

func (a *App) UpdateSettings(input codexswitch.UpdateSettingsInput) (codexswitch.AppState, error) {
	return a.service.UpdateSettings(input)
}
