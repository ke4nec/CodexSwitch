# CodexSwitch

CodexSwitch 是一个基于 `Wails + Go + Vue 3 + TypeScript + Vuetify + Pinia` 的桌面工具，用来管理多个 Codex 配置并在它们之间快速切换。

## 当前实现范围

- 自动扫描并同步当前 Codex 配置目录
- 识别官方配置和 API 配置
- 维护托管配置仓库
- 手动导入当前配置
- 新增、编辑、删除 API 配置
- 在托管配置之间切换
- 拉取并缓存官方配置额度信息
- 用桌面 UI 完成上述操作

## 环境要求

- Go 1.26+
- Node.js 22+
- Wails CLI

如果本机还没装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 批处理脚本

仓库根目录提供了纯 `bat` 脚本：

- `bootstrap.bat`
- `dev.bat`
- `build.bat`
- `release.bat`

## 常用用法

初始化依赖：

```cmd
bootstrap.bat
```

启动调试：

```cmd
dev.bat
```

只检查调试环境，不真正启动：

```cmd
dev.bat -CheckOnly
```

调试构建，但跳过 Wails 打包：

```cmd
build.bat -SkipWails
```

完整调试构建：

```cmd
build.bat
```

发布构建，但跳过 Wails 打包：

```cmd
release.bat -SkipWails
```

完整发布构建：

```cmd
release.bat
```

额外参数会透传给 `wails`：

```cmd
build.bat -clean
release.bat -platform windows/amd64
```

## 脚本行为

- `bootstrap.bat`
  - 执行 `go mod tidy`
  - 检查并安装 `frontend` 依赖
- `dev.bat`
  - 检查 Go / npm / Wails
  - 自动补齐前端依赖
  - 执行 `wails dev`
- `build.bat`
  - 执行 `go test ./...`
  - 执行 `go build ./...`
  - 执行 `frontend` 的 `npm run build`
  - 执行 `wails build`
- `release.bat`
  - 执行 `go test ./...`
  - 执行 `go build ./...`
  - 执行 `frontend` 的 `npm run build`
  - 执行 `wails build -clean`

## 直接命令

如果你想手工执行：

```bash
go test ./...
go build ./...
cd frontend
npm install
npm run build
cd ..
wails dev
```

## 产物目录

- 前端构建产物：`frontend/dist`
- Wails 桌面程序产物：`build/bin`

## 目录说明

- `internal/codexswitch`：后端服务和测试
- `frontend`：Vue 3 前端
- `conf`：样例配置
- `codext`：参考项目
