package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	nativeDialog "github.com/sqweek/dialog"
)

func main() {
	// 切换工作目录到可执行文件所在目录，使相对路径和文件对话框行为可预测
	if exePath, err := os.Executable(); err == nil {
		if exeDir := filepath.Dir(exePath); exeDir != "" {
			_ = os.Chdir(exeDir)
		}
	}

	a := app.NewWithID("yinr.cc.yt-dlp-simpgo")
	a.SetIcon(&fyne.StaticResource{
		StaticName:    "icon.png",
		StaticContent: iconData,
	})
	w := a.NewWindow("视频下载工具 " + Version)
	w.Resize(fyne.NewSize(720, 420))

	entry := widget.NewEntry()
	entry.SetPlaceHolder("在此输入视频网址，例如：https://...")

	iniPath := IniFileName
	defaultOut := filepath.Join(".", "下载")
	appCfg, ytdlpCfg, err := EnsureDefaults(iniPath, defaultOut)
	if err != nil {
		dialog.ShowError(fmt.Errorf("初始化配置失败: %v", err), nil)
		os.Exit(1)
	}
	exeDir, _ := os.Getwd()

	outputBinding := binding.NewString()
	_ = outputBinding.Set(appCfg.OutputDir)
	linkContainer := container.NewHBox()

	setOutputBtn := widget.NewButton("设置下载目录", func() {
		startDir := appCfg.OutputDir
		if !filepath.IsAbs(startDir) {
			startDir = filepath.Join(exeDir, startDir)
		}
		p, err := nativeDialog.Directory().Title("选择下载目录").SetStartDir(startDir).Browse()
		if err != nil || p == "" {
			return
		}
		if rel, rerr := filepath.Rel(exeDir, p); rerr == nil && !strings.HasPrefix(rel, "..") {
			appCfg.OutputDir = rel
		} else {
			appCfg.OutputDir = p
		}
		actualPath := appCfg.OutputDir
		if !filepath.IsAbs(actualPath) {
			actualPath = filepath.Join(exeDir, actualPath)
		}
		if err := os.MkdirAll(actualPath, 0755); err != nil {
			dialog.ShowError(fmt.Errorf("无法创建目录: %v", err), w)
			return
		}
		if err := outputBinding.Set(appCfg.OutputDir); err != nil {
			dialog.ShowError(fmt.Errorf("无法设置输出目录: %v", err), w)
			return
		}
		if err := SaveAppConfig(iniPath, appCfg); err != nil {
			dialog.ShowError(fmt.Errorf("无法保存配置: %v", err), w)
			return
		}
	})

	// 打开文件夹辅助函数
	openFolder := func(path string, fileURL *url.URL) {
		if runtime.GOOS == "windows" {
			if err := exec.Command("explorer", path).Start(); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "错误",
					Content: fmt.Sprintf("无法打开文件夹: %v", err),
				})
			}
		} else if fileURL != nil {
			if err := fyne.CurrentApp().OpenURL(fileURL); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "错误",
					Content: fmt.Sprintf("无法打开URL: %v", err),
				})
			}
		}
	}

	// 创建可点击的目录链接按钮
	makeLink := func(path string) fyne.CanvasObject {
		var fileURL string
		if runtime.GOOS == "windows" {
			fileURL = "file:///" + filepath.ToSlash(path)
		} else {
			fileURL = "file://" + filepath.ToSlash(path)
		}
		u, _ := url.Parse(fileURL)
		return widget.NewButton(path, func() { openFolder(path, u) })
	}

	linkContainer.Add(makeLink(appCfg.OutputDir))

	outputBinding.AddListener(binding.NewDataListener(func() {
		p, _ := outputBinding.Get()
		linkContainer.Objects = []fyne.CanvasObject{makeLink(p)}
		linkContainer.Refresh()
	}))

	// 日志区域
	logBinding := binding.NewString()
	logLabel := widget.NewLabelWithData(logBinding)
	logLabel.Wrapping = fyne.TextWrapWord
	logScroll := container.NewVScroll(logLabel)

	clearBtn := widget.NewButton("清空", func() {
		entry.SetText("")
	})

	updateBtn := widget.NewButton("更新 yt-dlp", nil)
	downloadBtn := widget.NewButton("开始下载", nil)

	var runningMu sync.Mutex
	running := false

	aboutBtn := widget.NewButton("关于", func() {
		content := widget.NewLabel(fmt.Sprintf("yt-dlp-simpgo %s\n\n"+
			"一个基于 yt-dlp 的简单图形界面下载工具\n\n"+
			"项目地址: "+Repository, Version))
		content.Wrapping = fyne.TextWrapWord
		scroll := container.NewScroll(content)
		scroll.SetMinSize(fyne.NewSize(400, 200))
		dialog.ShowCustom("关于", "关闭", scroll, w)
	})

	settingsBtn := widget.NewButton("设置", func() {
		showSettingsDialog(w, appCfg, ytdlpCfg, exeDir, func(newAppCfg *AppConfig, newYtdlpCfg *YTDLPConfig) {
			appCfg = newAppCfg
			ytdlpCfg = newYtdlpCfg
			if err := outputBinding.Set(appCfg.OutputDir); err != nil {
				dialog.ShowError(fmt.Errorf("无法更新输出目录: %v", err), w)
			}
			linkContainer.Objects = []fyne.CanvasObject{makeLink(appCfg.OutputDir)}
			linkContainer.Refresh()
		})
	})

	// 日志绑定自动滚动
	logBinding.AddListener(binding.NewDataListener(func() {
		logScroll.ScrollToBottom()
	}))

	addLog := func(text string, rewriteLast bool) {
		go func() {
			cur, err := logBinding.Get()
			if err != nil {
				return
			}
			if cur == "" {
				_ = logBinding.Set(text)
			} else if rewriteLast {
				idx := strings.LastIndex(cur, "\n")
				if idx == -1 {
					idx = 0
				}
				_ = logBinding.Set(cur[:idx] + "\n" + text)
			} else {
				_ = logBinding.Set(cur + "\n" + text)
			}
		}()
	}
	appendLog := func(text string) { addLog(text, false) }
	rewriteLog := func(text string) { addLog(text, true) }

	// 根据 yt-dlp 是否存在设置按钮行为
	ytDlpPath, found := findYtDlp(exeDir)
	if found {
		downloadBtn.SetText("开始下载")
		downloadBtn.OnTapped = func() {
			urlStr := strings.TrimSpace(entry.Text)
			if urlStr == "" {
				dialog.ShowInformation("提示", "请输入一个有效的网址", w)
				return
			}
			startDownload(ytDlpPath, urlStr, appCfg.OutputDir, exeDir, &runningMu, &running, appendLog)
		}
		wireUpdateBtn(updateBtn, ytDlpPath, appCfg.DownloadProxy, appendLog)
	} else {
		updateBtn.Disable()
		downloadBtn.SetText("下载 yt-dlp")
		downloadBtn.OnTapped = func() {
			appendLog("正在下载 yt-dlp 到: " + exeDir)
			appendLog("")
			go func() {
				var lastProgress string
				var lastPct = -1
				onProgress := func(received, total int64) {
					var progress string
					var pct int
					if total > 0 {
						pct = int(float64(received) * 100.0 / float64(total))
						progress = fmt.Sprintf("下载 yt-dlp: %d%% (%d/%d)", pct, received, total)
					} else {
						progress = fmt.Sprintf("下载 yt-dlp: %d bytes", received)
					}
					if (pct != lastPct) && (progress != lastProgress) {
						lastPct = pct
						lastProgress = progress
						fyne.Do(func() { rewriteLog(progress) })
					}
				}

				p, derr := DownloadYtDlpWithProgress(exeDir, appCfg.DownloadProxy, appCfg.YtDlpURL, onProgress)
				if derr != nil {
					fyne.Do(func() {
						appendLog("下载 yt-dlp 失败: " + derr.Error())
						dialog.ShowError(derr, w)
					})
					return
				}
				fyne.Do(func() {
					appendLog("已下载: " + p)
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已下载"})
					downloadBtn.SetText("开始下载")
					downloadBtn.OnTapped = func() {
						urlStr := strings.TrimSpace(entry.Text)
						if urlStr == "" {
							dialog.ShowInformation("提示", "请输入一个有效的网址", w)
							return
						}
						startDownload(p, urlStr, appCfg.OutputDir, exeDir, &runningMu, &running, appendLog)
					}
					wireUpdateBtn(updateBtn, p, appCfg.DownloadProxy, appendLog)
				})
			}()
		}
	}

	entryRow := container.NewBorder(nil, nil, nil, clearBtn, entry)
	buttons := container.NewHBox(setOutputBtn, widget.NewLabel("下载目录:"), linkContainer, layout.NewSpacer(), updateBtn, downloadBtn, settingsBtn, aboutBtn)
	top := container.NewVBox(entryRow, buttons)
	content := container.NewBorder(top, nil, nil, nil, logScroll)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}
