<p align="right">
  <a href="./README.md">简体中文</a> | <strong>English</strong>
</p>

# CodexSwitch

<p align="center">
  <img src="./build/appicon.png" alt="CodexSwitch Logo" width="120" />
</p>

<p align="center">
  <strong>Codex profile manager, Codex account switcher, and OpenAI API configuration desktop app</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Desktop-Wails%20App-2f855a?style=for-the-badge" alt="Desktop App" />
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26+" />
  <img src="https://img.shields.io/badge/Node.js-22.x-339933?style=for-the-badge&logo=node.js&logoColor=white" alt="Node.js 22.x" />
  <img src="https://img.shields.io/badge/Wails-v2.11.0-FF6B6B?style=for-the-badge" alt="Wails v2.11.0" />
  <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?style=for-the-badge&logo=vue.js&logoColor=white" alt="Vue 3.5" />
  <img src="https://img.shields.io/badge/TypeScript-5.8-3178C6?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript 5.8" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Platforms-Windows%20%7C%20macOS%20%7C%20Linux-1f2937?style=flat-square" alt="Platforms" />
  <img src="https://img.shields.io/badge/macOS-Intel%20%26%20Apple%20Silicon-111827?style=flat-square" alt="macOS Architectures" />
  <img src="https://img.shields.io/badge/Release-Automated-7c3aed?style=flat-square" alt="Automated Release" />
  <img src="https://img.shields.io/badge/Version-1.0.0%2B-2563eb?style=flat-square" alt="Versioning" />
  <img src="https://img.shields.io/badge/Language-English-3b82f6?style=flat-square" alt="Language English" />
</p>

<p align="center">
  <a href="https://github.com/ke4nec/CodexSwitch/releases"><img src="https://img.shields.io/github/v/release/ke4nec/CodexSwitch?style=flat-square" alt="Latest Release" /></a>
  <a href="https://github.com/ke4nec/CodexSwitch/stargazers"><img src="https://img.shields.io/github/stars/ke4nec/CodexSwitch?style=flat-square" alt="GitHub Stars" /></a>
  <a href="https://github.com/ke4nec/CodexSwitch/issues"><img src="https://img.shields.io/github/issues/ke4nec/CodexSwitch?style=flat-square" alt="GitHub Issues" /></a>
  <a href="https://github.com/ke4nec/CodexSwitch/releases"><img src="https://img.shields.io/github/downloads/ke4nec/CodexSwitch/total?style=flat-square" alt="GitHub Downloads" /></a>
</p>

<p align="center">
  <a href="https://github.com/ke4nec/CodexSwitch/releases">Download</a> ·
  <a href="./CONTRIBUTING.md">Contributing</a>
</p>

CodexSwitch is a cross-platform desktop application for managing multiple Codex configurations.

It connects your active Codex home directory with a locally managed profile library, so you can switch between official accounts and custom API setups without manually editing `auth.json` or `config.toml`.

## Preview

<p align="center">
  <img src="./docs/preview.png" alt="CodexSwitch Preview" width="100%" />
</p>

If you are searching for any of the following, this project is likely relevant:
- Codex profile manager
- Codex account switcher
- Codex multi-account desktop app
- OpenAI API profile manager
- cross-platform Codex desktop app for Windows, macOS, and Linux

---

## Why CodexSwitch

CodexSwitch is useful when you need to:

- switch between multiple official Codex accounts
- move between official accounts and OpenAI API profiles
- maintain several `API Key`, model, and reasoning-effort combinations
- quickly inspect quota, status, and availability
- avoid repeated manual edits inside `~/.codex`

### Who It Is For

- individual users switching between multiple official Codex accounts
- developers managing both official Codex access and OpenAI API keys
- desktop users looking for a visual Codex configuration switcher
- teams that want the same workflow across Windows, macOS, and Linux

### Search Keywords

- CodexSwitch
- Codex profile manager
- Codex account switcher
- Codex desktop app
- OpenAI API profile manager
- multi-account manager for Codex
- Wails desktop app

---

## Highlights

- **Official account import**
  Automatically detects the current Codex profile and also supports importing exported official account files from the UI.

- **API profile management**
  Create, edit, store, and switch between multiple API-based profiles.

- **One-click switching**
  Apply the selected profile back to the target Codex directory without hand-editing config files.

- **Managed profile library**
  Keep your commonly used profiles in one local managed repository.

- **Rate limit refresh**
  Fetch and cache official account usage windows for quick inspection.

- **Latency and availability testing**
  Run responsiveness checks for both official and API profiles.

- **Cross-platform desktop app**
  Built with Wails, Go, and Vue for Windows, macOS, and Linux.

- **Bilingual UI**
  The application supports both Chinese and English UI languages.

---

## How It Works

The core flow is straightforward:

1. Scan the configured Codex home directory
2. Detect whether the current setup is an official or API profile
3. Store recognized profiles in a managed local library
4. Let you import, edit, switch, test, and remove profiles in the UI
5. Write the selected profile back to the active Codex directory

---

## Quick Start

### Typical Usage

1. Launch the app
2. Open Settings and confirm the target Codex home directory
3. Let the app detect the current profile automatically, or import an official account file
4. Add API profiles when needed
5. Switch profiles, refresh limits, or run latency tests from the list

### Typical Codex Home Locations

- macOS / Linux: `~/.codex`
- Windows: usually `%USERPROFILE%\.codex`
- The path can also be changed in the app settings

---

## Download & Releases

- The project automatically produces release assets for:
  - Linux `amd64`
  - Windows `amd64`
  - macOS `amd64`
  - macOS `arm64`
- Official builds are uploaded to the GitHub **Releases** page
- These are user-facing release assets, not only temporary workflow artifacts

### Versioning

- The project version is stored in [`wails.json`](wails.json) under `info.productVersion`
- You can start from the default version `1.0.0`
- Releases no longer auto-increment on every commit
- If the tag for the current version already exists, the workflow fails instead of overwriting an existing release

### Two Ways To Trigger A Release

- Update `info.productVersion` in [`wails.json`](wails.json) and push to `master`
- Push a version tag such as `v1.0.1`

### Recommended Release Flow

1. Update `info.productVersion` in [`wails.json`](wails.json)
2. Commit and push to `master`
3. GitHub Actions builds and publishes that version automatically

If you prefer explicit tag-based releases, you can also do:

```bash
git tag v1.0.1
git push origin v1.0.1
```

### Release Workflow

- Workflow file: [`.github/workflows/release-cross-platform.yml`](.github/workflows/release-cross-platform.yml)
- Trigger:
  - push to `master` when [`wails.json`](wails.json) changes
  - push a `v*` version tag
  - manual dispatch as a fallback

---

## Build From Source

### Requirements

- Go `1.26+`
- Node.js `22.x`
- Wails CLI `v2.11.0+`

Install the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Bootstrap

```cmd
bootstrap.bat
```

### Development

```cmd
dev.bat
```

### Local Build

```cmd
build.bat
```

### Local Release Build

```cmd
release.bat
```

### Clean

```cmd
clean.bat
```

Remove all generated files including `node_modules`:

```cmd
clean.bat -All
```

---

## Manual Commands

If you prefer running commands directly:

```bash
cd frontend
npm install
npm run build
cd ..
go test ./...
go build ./...
wails build
```

---

## Tech Stack

- **Backend**: Go
- **Desktop Shell**: Wails v2
- **Frontend**: Vue 3 + TypeScript + Vuetify + Pinia
- **Build & Release**: GitHub Actions

---

## Project Structure

- [`internal/codexswitch`](internal/codexswitch): backend services, storage, parsing, tests
- [`frontend`](frontend): Vue desktop UI
- [`conf`](conf): sample configuration data
- [`build`](build): packaging resources and build output

---

## Notes

- Official and API profiles are intentionally handled through different internal flows.
- The app tries to keep a single stable Codex directory and switch its content, instead of making you manage multiple folders manually.
- macOS releases are generated for both Intel and Apple Silicon.
- Linux release builds use the Wails `webkit2_41` build tag for Ubuntu 24.04 compatibility.

---

## Current Focus

This project is currently focused on practical profile management and desktop usability:

- unified management of official and API profiles
- automatic detection and write-back of the active Codex home
- official quota refresh
- latency and availability checks
- automated cross-platform releases

---

## Acknowledgements

CodexSwitch is built on top of the Wails ecosystem and the broader Go + Vue open-source stack.
