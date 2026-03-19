# CodexSwitch

CodexSwitch 是一个基于 `Wails + Go + Vue 3 + TypeScript + Vuetify + Pinia` 的桌面工具，用来管理多个 Codex 配置并在它们之间快速切换。

## 当前实现范围

- 自动扫描并同步当前 `Codex` 配置目录
- 识别官方配置和 API 配置
- 托管配置仓库读写
- 手动导入当前配置
- 新增、编辑、删除 API 配置
- 在托管配置之间切换
- 官方配置额度信息拉取与缓存
- 基于 Vuetify 的单页管理界面

## 本地开发

1. 安装 Go 1.26+ 和 Node.js 22+
2. 安装 Wails CLI
3. 在仓库根目录执行：

```bash
go mod tidy
cd frontend
npm install
npm run build
cd ..
wails dev
```

如果只跑 Go 测试：

```bash
go test ./internal/codexswitch
```

## 目录说明

- `internal/codexswitch`：核心后端服务与测试
- `frontend`：Vue 3 前端
- `conf`：配置样例
- `codext`：参考项目
