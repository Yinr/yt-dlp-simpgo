# AGENTS.md - yt-dlp-simpgo

<!-- Generated: 2026-06-08 | Commit: 2cf4ef4 | Branch: main -->

基于 [yt-dlp](https://github.com/yt-dlp/yt-dlp) 的 Go 语言图形界面下载工具，使用 [Fyne](https://fyne.io/) 框架。模块路径：`yinr.cc/yt-dlp-simpgo`。Go 版本：1.25。

## STRUCTURE

```
main.go             - 程序入口、主窗口 UI 布局、启动时版本检测提示
config.go           - AppConfig / YTDLPConfig 类型定义、INI+conf 配置持久化、go:embed 嵌入默认资源
config_test.go      - 配置解析/保存的单元测试
download.go         - 下载辅助函数：findYtDlp、startDownload、wireUpdateBtn
progress.go         - ProgressReporter 结构体及方法（日志/进度条/重写回调）
process_output.go   - 通用子进程输出读取和编码转换（decodeProcessLine、readPipe）
yt_dlp_output.go    - yt-dlp 输出解析（YtDlpEvent 结构体、friendlyYtDlpLine、readYtDlpPipe）
format.go           - 通用格式化辅助函数（formatBytes、logMarker）
format_test.go      - 格式化辅助函数单元测试
settings.go         - 设置对话框 UI
self_update.go      - 程序自身更新检测、下载、校验、替换和重启
self_update_test.go - 程序自身更新相关单元测试
download_test.go    - friendlyYtDlpLine 单元测试
yt_dlp.go           - 下载/更新/版本检测 yt-dlp（HTTP 代理、进度回调、GitHub API）
version.go          - 程序版本号和仓库地址常量（Version 由 ldflags 注入）
utils/              - 平台相关辅助函数及测试（execCmd_win.go, execCmd_nowin.go, execCmd_test.go）
res/                - 嵌入资源（窗口图标、Windows exe 图标、默认配置文件）
dist/               - 构建产物输出目录
rsrc_windows_*.syso - Windows exe 图标资源中间产物，构建时自动生成，不提交
```

## WHERE TO LOOK

| 需求 | 文件 |
|---|---|
| 主窗口 UI / 启动逻辑 | `main.go` |
| 配置读写 / 嵌入资源 | `config.go` |
| 下载流程控制 | `download.go` |
| 进度/日志回调 | `progress.go` |
| yt-dlp 输出解析 | `yt_dlp_output.go` |
| 设置对话框 | `settings.go` |
| 自身更新逻辑 | `self_update.go` |
| 平台相关命令执行 | `utils/execCmd_*.go` |

## CODE MAP

- `main.go`：创建 Fyne 应用和主窗口，初始化配置，绑定下载按钮事件，启动时检查程序自身和 yt-dlp 版本更新。
- `config.go`：`AppConfig`（输出目录、代理、yt-dlp URL）和 `YTDLPConfig`（字幕、章节、元数据、合并格式等）的加载/保存；通过 `//go:embed` 嵌入 `res/` 中的默认配置和图标。
- `download.go`：查找 `yt-dlp` 可执行文件，构造命令行参数，启动下载进程。
- `progress.go`：`ProgressReporter` 处理 yt-dlp 的进度事件，更新 UI 进度条和日志。
- `yt_dlp.go`：HTTP 代理设置，GitHub API 查询最新版本，下载和替换 yt-dlp 二进制。
- `self_update.go`：查询 GitHub Releases，下载新版本，校验校验和，替换当前可执行文件并重启。

## CONVENTIONS

### 导入分组
- 标准库 → 外部依赖 → 内部包（`yinr.cc/yt-dlp-simpgo/utils`）
- 组间空行分隔；名称冲突时使用别名（如 `nativeDialog "github.com/sqweek/dialog"`、`ini "gopkg.in/ini.v1"`）

### 命名
- 导出类型：`PascalCase`（`AppConfig`、`YTDLPConfig`、`ExecCmd`）
- 非导出函数：`camelCase`（`writeUTF8BOMFile`、`findYtDlp`、`startDownload`）
- 配置常量：`PascalCase`（`IniFileName`、`YTDLPConfName`）
- 文件名：`snake_case`（`yt_dlp.go`、`execCmd_win.go`）

### 平台相关代码
- 构建标签：`//go:build windows` / `//go:build !windows`
- 文件命名：`_win.go`（Windows）/ `_nowin.go`（非 Windows）

### 错误处理
- 包装错误：`fmt.Errorf("上下文: %w", err)`
- 用户可见错误信息使用中文（如 `"无法创建目录: %w"`）
- 非关键错误用 `_ =` 忽略（如 `_ = os.Chdir(exeDir)`）
- UI 错误报告：`dialog.ShowError(...)` 或 `fyne.CurrentApp().SendNotification(...)`

### 并发
- `sync.Mutex` 保护共享状态（如 `runningMu` 守护 `running`）
- `sync.WaitGroup` 等待 goroutine 完成
- goroutine 中更新 UI 必须使用 `fyne.Do(func() { ... })`

### 嵌入资源
- `//go:embed` 指令放在 `config.go`，与对应 `var` 声明放在一起
- Windows exe 文件图标通过 `rsrc_windows_*.syso` 在构建期嵌入，非 `a.SetIcon()`

### 格式化
- `gofmt` 标准格式（tab 缩进）
- Fyne 容器布局：`HBox`、`VBox`、`Border` 组合 UI
- 小型事件处理器和 UI 回调可接受内联闭包

## ANTI-PATTERNS

- 不要在 goroutine 中直接调用 Fyne UI 更新方法，必须用 `fyne.Do`
- 不要忽略构建标签，平台相关代码必须成对出现（`_win.go` + `_nowin.go`）
- 不要将 `rsrc_windows_*.syso` 提交到版本控制（已在 `.gitignore` 中）
- 不要在用户可见的错误中使用英文（日志/调试信息除外）

## COMMANDS

```bash
# 构建
make build        # 当前平台 → dist/
make build-gui    # Windows GUI 版本 → dist/
make release      # 多平台版本 → dist/
make clean        # 清理构建产物
make clean-runtime # 清理运行时文件（配置、下载目录、yt-dlp）
make clean-all    # 清理构建产物和运行时文件

# 测试
make test                                 # 运行所有测试（make 封装）
go test -v ./...                          # 运行所有测试
go test -v ./utils                        # 测试指定包
go test -v -run TestFunctionName ./...     # 按名称运行单个测试
go test -v -count=1 ./...                 # 强制重新运行（跳过缓存）
go test -cover ./...                       # 查看测试覆盖率

# PowerShell 构建（Windows）
.\build.ps1                               # 构建当前平台
.\build.ps1 -Gui                          # Windows GUI 版本
.\build.ps1 -Clean                        # 清理构建产物
.\build.ps1 -CleanRuntime                 # 清理运行时文件
.\build.ps1 -CleanAll                     # 清理构建产物和运行时文件

# 依赖
make install-deps                         # go mod tidy
```

## NOTES

- Windows GUI 构建使用固定版本 `github.com/akavel/rsrc@v0.10.2` 和 `res/icon.ico` 生成 `rsrc_windows_amd64.syso`
- 若 `dist/yt-dlp-simpgo.exe` 正在运行，Windows 会锁定文件，重新构建前应先关闭旧进程
- 无第三方 linter 配置；遵循标准 Go 惯例（`go vet`、`gofmt`）
- 用户可见的注释和提示文本使用中文；代码注释可使用英文
- 自解释的代码不添加注释
