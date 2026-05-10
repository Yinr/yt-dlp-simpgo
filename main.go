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
	progressLabel := widget.NewLabel("就绪")
	progressBar := widget.NewProgressBar()

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
	setProgress := func(status string, value float64) {
		fyne.Do(func() {
			progressLabel.SetText(status)
			if value < 0 {
				progressBar.SetValue(0)
				return
			}
			if value > 1 {
				value = 1
			}
			progressBar.SetValue(value)
		})
	}
	clearProgress := func() {
		fyne.Do(func() {
			progressLabel.SetText("就绪")
			progressBar.SetValue(0)
		})
	}
	reporter := ProgressReporter{
		AppendLog:   appendLog,
		RewriteLog:  rewriteLog,
		SetProgress: setProgress,
		Clear:       clearProgress,
	}

	go func() {
		info, err := CheckAppUpdate(Version, appCfg.DownloadProxy)
		if err != nil || info == nil {
			return
		}

		fyne.Do(func() {
			appendLog(logMarker("程序更新可用"))
			appendLog(fmt.Sprintf("程序有新版本: %s -> %s", info.CurrentVersion, info.LatestVersion))
			dialog.ShowConfirm("程序更新可用",
				fmt.Sprintf("检测到 yt-dlp-simpgo 新版本 %s（当前 %s），是否立即更新并重启？", info.LatestVersion, info.CurrentVersion),
				func(ok bool) {
					if !ok {
						return
					}
					go func() {
						appendLog(logMarker("开始下载程序更新"))
						appendLog("正在下载程序更新: " + info.AssetName)
						setProgress("正在下载程序更新", 0)
						var lastProgress string
						var lastPct = -1
						onProgress := func(received, total int64) {
							progress, pct := formatProgress("下载程序更新", received, total)
							if pct != lastPct || progress != lastProgress {
								lastPct = pct
								lastProgress = progress
								setProgress(progress, float64(pct)/100)
								fyne.Do(func() { rewriteLog(progress) })
							}
						}

						updatePath, err := DownloadAppUpdate(info, appCfg.DownloadProxy, onProgress)
						if err != nil {
							fyne.Do(func() {
								appendLog(logMarker("程序更新下载失败"))
								appendLog("程序更新下载失败: " + err.Error())
								clearProgress()
								dialog.ShowError(err, w)
							})
							return
						}

						if err := InstallAppUpdateAndRestart(updatePath); err != nil {
							fyne.Do(func() {
								appendLog(logMarker("程序更新安装失败"))
								appendLog("程序更新安装失败: " + err.Error())
								clearProgress()
								dialog.ShowError(err, w)
							})
							return
						}

						fyne.Do(func() {
							appendLog(logMarker("程序更新已安装"))
							appendLog("正在重启")
							a.Quit()
						})
					}()
				}, w)
		})
	}()

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
			startDownload(ytDlpPath, urlStr, appCfg.OutputDir, exeDir, &runningMu, &running, reporter)
		}
		wireUpdateBtn(updateBtn, ytDlpPath, appCfg.DownloadProxy, reporter)

		// 后台检查 yt-dlp 版本
		go func() {
			current, latest, needUpdate, err := CheckYtDlpUpdate(ytDlpPath, appCfg.DownloadProxy)
			if err != nil {
				// 版本检查失败不影响正常使用
				return
			}
			if needUpdate {
				fyne.Do(func() {
					appendLog(fmt.Sprintf("yt-dlp 有新版本: %s -> %s", current, latest))
					dialog.ShowConfirm("更新可用",
						fmt.Sprintf("检测到 yt-dlp 新版本 %s（当前 %s），是否更新？", latest, current),
						func(ok bool) {
							if ok {
								updateBtn.OnTapped()
							}
						}, w)
				})
			}
		}()
	} else {
		updateBtn.Disable()
		downloadBtn.SetText("下载 yt-dlp")
		downloadBtn.OnTapped = func() {
			appendLog(logMarker("开始下载 yt-dlp"))
			appendLog("正在下载 yt-dlp 到: " + exeDir)
			appendLog("")
			setProgress("正在下载 yt-dlp", 0)
			go func() {
				var lastProgress string
				var lastPct = -1
				onProgress := func(received, total int64) {
					progress, pct := formatProgress("下载 yt-dlp", received, total)
					if (pct != lastPct) && (progress != lastProgress) {
						lastPct = pct
						lastProgress = progress
						if total > 0 {
							setProgress(progress, float64(pct)/100)
						} else {
							setProgress(progress, 0)
						}
						fyne.Do(func() { rewriteLog(progress) })
					}
				}

				p, derr := DownloadYtDlpWithProgress(exeDir, appCfg.DownloadProxy, appCfg.YtDlpURL, onProgress)
				if derr != nil {
					fyne.Do(func() {
						appendLog(logMarker("下载 yt-dlp 失败"))
						appendLog("下载 yt-dlp 失败: " + derr.Error())
						clearProgress()
						dialog.ShowError(derr, w)
					})
					return
				}
				fyne.Do(func() {
					setProgress("yt-dlp 下载完成", 1)
					appendLog(logMarker("yt-dlp 下载完成"))
					appendLog("已下载: " + p)
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已下载"})
					downloadBtn.SetText("开始下载")
					downloadBtn.OnTapped = func() {
						urlStr := strings.TrimSpace(entry.Text)
						if urlStr == "" {
							dialog.ShowInformation("提示", "请输入一个有效的网址", w)
							return
						}
						startDownload(p, urlStr, appCfg.OutputDir, exeDir, &runningMu, &running, reporter)
					}
					wireUpdateBtn(updateBtn, p, appCfg.DownloadProxy, reporter)
				})
			}()
		}
	}

	entryRow := container.NewBorder(nil, nil, nil, clearBtn, entry)
	buttons := container.NewHBox(setOutputBtn, widget.NewLabel("下载目录:"), linkContainer, layout.NewSpacer(), updateBtn, downloadBtn, settingsBtn, aboutBtn)
	progressBox := container.NewVBox(progressLabel, progressBar)
	top := container.NewVBox(entryRow, buttons, progressBox)
	content := container.NewBorder(top, nil, nil, nil, logScroll)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}
