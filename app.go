package main

import (
	"context"
	"log/slog"
	"os"

	"codexswitch/internal/codexswitch"
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
