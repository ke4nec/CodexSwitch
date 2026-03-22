package main

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed wails.json
var wailsConfigRaw []byte

var appVersion = parseAppVersion(wailsConfigRaw)

func parseAppVersion(raw []byte) string {
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}

	if err := json.Unmarshal(raw, &config); err != nil {
		return "unknown"
	}

	version := strings.TrimSpace(config.Info.ProductVersion)
	if version == "" {
		return "unknown"
	}

	return version
}
