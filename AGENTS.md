# AGENTS.md - yt-dlp-simpgo

基于 [yt-dlp](https://github.com/yt-dlp/yt-dlp) 的 Go 语言图形界面下载工具，使用 [Fyne](https://fyne.io/) 框架。模块路径：`yinr.cc/yt-dlp-simpgo`。Go 版本：1.25。

## 项目结构

```
main.go          - 程序入口、主窗口 UI 布局、启动时程序自身和 yt-dlp 版本检测提示
config.go        - AppConfig / YTDLPConfig 类型定义、INI+conf 配置持久化、go:embed 嵌入默认资源
config_test.go   - 配置解析/保存的单元测试
download.go      - 下载辅助函数：findYtDlp、readPipe、startDownload、wireUpdateBtn、yt-dlp 输出解析
format.go        - 通用格式化辅助函数（如自动单位文件大小显示）
settings.go      - 设置对话框 UI
self_update.go   - 程序自身更新检测、下载、校验、替换和重启
self_update_test.go - 程序自身更新相关单元测试
yt_dlp.go        - 下载/更新/版本检测 yt-dlp（HTTP 代理、进度回调、GitHub API）
version.go       - 程序版本号和仓库地址常量（Version 由 ldflags 注入）
utils/           - 平台相关辅助函数及测试（execCmd_win.go, execCmd_nowin.go, execCmd_test.go）
res/             - 嵌入资源（窗口图标、Windows exe 图标、默认配置文件）
dist/            - 构建产物输出目录
rsrc_windows_*.syso - Windows exe 图标资源中间产物，构建时自动生成，不提交
```

## 构建命令

```bash
make build                 # 构建当前平台版本 → dist/
make build-gui             # 构建 Windows GUI 版本 → dist/
make release               # 构建多平台版本 → dist/
make clean                 # 清理构建产物（dist/、rsrc_windows_*.syso）
make clean-runtime         # 清理运行时文件（配置、下载目录、yt-dlp）
make clean-all             # 清理构建产物和运行时文件
make test                  # 运行全部测试：go test -v ./...
make install-deps          # go mod tidy

.\build.ps1                # PowerShell 构建 → dist/
.\build.ps1 -Gui           # PowerShell GUI 构建 → dist/
.\build.ps1 -Clean         # PowerShell 清理构建产物
.\build.ps1 -CleanRuntime  # PowerShell 清理运行时文件
.\build.ps1 -CleanAll      # PowerShell 清理构建产物和运行时文件
```

Windows GUI 构建会先用固定版本的 `github.com/akavel/rsrc@v0.10.2` 和 `res/icon.ico` 生成 `rsrc_windows_amd64.syso`，从而把文件图标嵌入 exe；`rsrc_windows_*.syso` 是中间产物，已在 `.gitignore` 中忽略。若 `dist/yt-dlp-simpgo.exe` 正在运行，Windows 会锁定文件，重新构建前应先关闭旧进程。

## 测试

```bash
go test -v ./...                        # 运行所有测试
go test -v ./utils                      # 测试指定包
go test -v -run TestFunctionName ./...   # 按名称运行单个测试
go test -v -count=1 ./...               # 强制重新运行（跳过缓存）
go test -cover ./...                     # 查看测试覆盖率
```

添加测试时，请在被测代码同目录下创建 `*_test.go` 文件（例如根目录下 `config_test.go`，`utils/` 下 `execCmd_test.go`）。

## 代码风格

### 导入
- 按顺序分组导入：标准库、外部依赖、内部包（`yinr.cc/yt-dlp-simpgo/utils`）
- 组间用空行分隔。名称冲突时使用导入别名（如 `nativeDialog "github.com/sqweek/dialog"`、`ini "gopkg.in/ini.v1"`）

### 命名规范
- 导出类型使用 PascalCase：`AppConfig`、`YTDLPConfig`、`ExecCmd`
- 非导出函数使用 camelCase：`writeUTF8BOMFile`、`findYtDlp`、`startDownload`
- 配置相关常量使用 UPPER_SNAKE_CASE：`IniFileName`、`YTDLPConfName`
- 文件名使用 snake_case：`yt_dlp.go`、`execCmd_win.go`

### 平台相关代码
- 使用 Go 构建标签（`//go:build windows` / `//go:build !windows`）在成对文件中区分平台
- 文件命名约定：`_win.go`（Windows）/ `_nowin.go`（非 Windows）

### 错误处理
- 使用 `%w` 包装错误：`fmt.Errorf("上下文信息: %w", err)`
- 用户可见的错误信息使用中文（如 `"无法创建目录: %w"`）
- 非关键错误用 `_ =` 忽略（如 `_ = os.Chdir(exeDir)`）
- 通过 `dialog.ShowError(...)` 或 `fyne.CurrentApp().SendNotification(...)` 向 UI 报告错误

### 并发
- 使用 `sync.Mutex` 保护共享状态（如 `runningMu` 守护 `running` 标志位）
- 使用 `sync.WaitGroup` 等待 goroutine 完成
- 从 goroutine 调用 UI 更新时使用 `fyne.Do(func() { ... })` 将调用分发到 UI 线程

### 嵌入资源
- 使用 `//go:embed` 嵌入静态资源（图标、默认配置文件）
- 嵌入指令放在 `config.go` 中，与对应的 `var` 声明放在一起
- Windows exe 文件图标不由 `a.SetIcon()` 控制，需通过 `rsrc_windows_*.syso` 在构建期嵌入

### 格式化
- 使用 `gofmt`（tab 缩进，标准 Go 格式）
- 保持行长度合理；使用 Fyne 容器布局（HBox、VBox、Border）组合 UI
- 小型事件处理器和 UI 回调可接受内联闭包

### 通用规则
- 无第三方 linter 配置；遵循标准 Go 惯例（`go vet`、`gofmt`）
- 用户可见的注释和提示文本使用中文；代码注释可使用英文
- 自解释的代码不添加注释
