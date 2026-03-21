package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"

	"codexswitch/internal/codexswitch"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	service *codexswitch.Service
	mu      sync.Mutex
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
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

func (a *App) shutdown(context.Context) {}

func (a *App) GetAppState() (codexswitch.AppState, error) {
	return withAppLock(&a.mu, a.service.GetAppState)
}

func (a *App) ImportCurrentProfile() (codexswitch.AppState, error) {
	return withAppLock(&a.mu, a.service.ImportCurrentProfile)
}

func (a *App) ImportOfficialProfileFile() (codexswitch.AppState, error) {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()

	if ctx == nil {
		return codexswitch.AppState{}, errors.New("Wails runtime 未就绪")
	}

	selectedFile, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
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

	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.ImportOfficialProfileFile(selectedFile)
	})
}

func (a *App) CreateApiProfile(input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.CreateAPIProfile(input)
	})
}

func (a *App) UpdateApiProfile(id string, input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.UpdateAPIProfile(id, input)
	})
}

func (a *App) GetApiProfileInput(id string) (codexswitch.APIProfileInput, error) {
	return withAppLock(&a.mu, func() (codexswitch.APIProfileInput, error) {
		return a.service.GetAPIProfileInput(id)
	})
}

func (a *App) SwitchProfile(id string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.SwitchProfile(id)
	})
}

func (a *App) DeleteProfile(id string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.DeleteProfile(id)
	})
}

func (a *App) RefreshRateLimits(ids []string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.RefreshRateLimits(ids)
	})
}

func (a *App) RefreshApiLatencyTests(ids []string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.RefreshAPILatencyTests(ids)
	})
}

func (a *App) UpdateSettings(input codexswitch.UpdateSettingsInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.UpdateSettings(input)
	})
}

func withAppLock[T any](mu *sync.Mutex, fn func() (T, error)) (T, error) {
	mu.Lock()
	defer mu.Unlock()
	return fn()
}
