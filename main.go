//go:generate goversioninfo -icon=res/icon.ico -manifest=res/yt-dlp-simpgo.exe.manifest -gofile=res/versioninfo.json
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	nativeDialog "github.com/sqweek/dialog"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func main() {
	// ensure current working directory is the executable directory to make
	// relative paths and dialogs behave predictably when double-clicking the exe.
	if exePath, err := os.Executable(); err == nil {
		if exeDir := filepath.Dir(exePath); exeDir != "" {
			_ = os.Chdir(exeDir) // best-effort, ignore error
		}
	}

	// create app with a stable unique ID so Preferences API works without error
	a := app.NewWithID("yinr.cc.yt-dlp-simpgo")
	a.SetIcon(&fyne.StaticResource{
		StaticName:    "icon.png",
		StaticContent: iconData,
	})
	w := a.NewWindow("视频下载工具 " + Version)
	w.Resize(fyne.NewSize(720, 420))

	// URL entry (single-line but taller for easier editing)
	entry := widget.NewEntry()
	entry.SetPlaceHolder("在此输入视频网址，例如：https://...")

	// output directory (default ./下载)
	iniPath := IniFileName
	defaultOut := filepath.Join(".", "下载")
	// Ensure default files exist and obtain effective outputDir, proxy and yt-dlp url
	outputDir, downloadProxy, ytDlpURL, _ := EnsureDefaults(iniPath, defaultOut)
	// after chdir above, the program directory is the current working directory
	exeDir, _ := os.Getwd()
	outputBinding := binding.NewString()
	_ = outputBinding.Set(outputDir)
	// create a small container to hold the clickable output link
	linkContainer := container.NewHBox()
	setOutputBtn := widget.NewButton("设置下载目录", func() {
		// determine starting directory: resolve configured outputDir (may be relative)
		startDir := outputDir
		if !filepath.IsAbs(startDir) {
			startDir = filepath.Join(exeDir, startDir)
		}
		// open native folder chooser with start directory
		p, err := nativeDialog.Directory().Title("选择下载目录").SetStartDir(startDir).Browse()
		if err != nil {
			// user cancelled or error; do nothing
			return
		}
		if p == "" {
			return
		}
		// if selected path is a subpath of the executable dir, store relative path
		if rel, rerr := filepath.Rel(exeDir, p); rerr == nil && !strings.HasPrefix(rel, "..") {
			outputDir = rel
		} else {
			outputDir = p
		}
		// make sure the actual directory exists (resolve relative against exeDir)
		actualPath := outputDir
		if !filepath.IsAbs(actualPath) {
			actualPath = filepath.Join(exeDir, actualPath)
		}
		if err := os.MkdirAll(actualPath, 0755); err != nil {
			dialog.ShowError(fmt.Errorf("无法创建目录: %v", err), w)
			return
		}
		if err := outputBinding.Set(outputDir); err != nil {
			dialog.ShowError(fmt.Errorf("无法设置输出目录: %v", err), w)
			return
		}
		// save config with new outputDir (yt-dlp.conf is kept as a separate file)
		if err := SaveConfig(iniPath, outputDir, downloadProxy, ytDlpURL); err != nil {
			dialog.ShowError(fmt.Errorf("无法保存配置: %v", err), w)
			return
		}
	})

	// helper to open folder (platform-specific)
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

	// helper to create a clickable link/button for a folder path
	makeLink := func(path string) fyne.CanvasObject {
		// build platform-correct file:// URL
		var fileURL string
		if runtime.GOOS == "windows" {
			fileURL = "file:///" + filepath.ToSlash(path)
		} else {
			fileURL = "file://" + filepath.ToSlash(path)
		}
		u, _ := url.Parse(fileURL)

		btn := widget.NewButton(path, func() { openFolder(path, u) })
		return btn
	}

	// initialize link container with current path
	linkContainer.Add(makeLink(outputDir))

	// when outputBinding changes, update the hyperlink
	outputBinding.AddListener(binding.NewDataListener(func() {
		p, _ := outputBinding.Get()
		linkContainer.Objects = []fyne.CanvasObject{makeLink(p)}
		linkContainer.Refresh()
	}))

	// (progress bar removed)

	// log area: use a Label bound to a string so text color stays normal,
	// then put it into a scroll container so it can grow.
	logBinding := binding.NewString()
	logLabel := widget.NewLabelWithData(logBinding)
	logLabel.Wrapping = fyne.TextWrapWord
	logScroll := container.NewVScroll(logLabel)

	clearBtn := widget.NewButton("清空", func() {
		entry.SetText("")
	})

	updateBtn := widget.NewButton("更新 yt-dlp", nil)
	downloadBtn := widget.NewButton("开始下载", nil)

	// thread-safe running flag to prevent re-entrancy
	var runningMu sync.Mutex
	running := false

	// About button
	aboutBtn := widget.NewButton("关于", func() {
		content := widget.NewLabel(fmt.Sprintf("yt-dlp-simpgo %s\n\n"+
			"一个基于 yt-dlp 的简单图形界面下载工具\n\n"+
			"项目地址: "+Repository, Version))
		content.Wrapping = fyne.TextWrapWord
		scroll := container.NewScroll(content)
		scroll.SetMinSize(fyne.NewSize(400, 200))
		dialog.ShowCustom("关于", "关闭", scroll, w)
	})

	// helper to append to log on main thread
	// add a listener so when binding changes we auto-scroll; binding listeners
	// are invoked on the UI thread so calling ScrollToBottom is safe here.
	logBinding.AddListener(binding.NewDataListener(func() {
		logScroll.ScrollToBottom()
	}))

	addLog := func(text string, rewriteLast bool) {
		// append via binding (get current, set new)
		go func() {
			cur, err := logBinding.Get()
			if err != nil {
				return
			}
			if cur == "" {
				_ = logBinding.Set(text)
			} else {
				if rewriteLast {
					curLastWrap := strings.LastIndex(cur, "\n")
					if curLastWrap == -1 {
						curLastWrap = 0
					}
					curWithoutLastLine := cur[:curLastWrap]
					_ = logBinding.Set(curWithoutLastLine + "\n" + text)
				} else {
					_ = logBinding.Set(cur + "\n" + text)
				}
			}
		}()
	}
	appendLog := func(text string) {
		addLog(text, false)
	}
	rewriteLog := func(text string) {
		addLog(text, true)
	}

	// download action will be set based on whether yt-dlp exists
	// helper to check for yt-dlp existence in exe dir or PATH
	findYtDlp := func() (string, bool) {
		exeName := "yt-dlp"
		if runtime.GOOS == "windows" {
			exeName = "yt-dlp.exe"
		}
		// check exe dir
		if p := filepath.Join(exeDir, exeName); func() bool { _, e := os.Stat(p); return e == nil }() {
			return p, true
		}
		// check PATH
		if pathP, err := exec.LookPath(exeName); err == nil {
			return pathP, true
		}
		return "", false
	}

	startDownload := func(exePath string) {
		url := strings.TrimSpace(entry.Text)
		if url == "" {
			dialog.ShowInformation("提示", "请输入一个有效的网址", w)
			return
		}

		// prevent double start
		runningMu.Lock()
		if running {
			runningMu.Unlock()
			dialog.ShowInformation("提示", "已有下载在进行中", w)
			return
		}
		running = true
		runningMu.Unlock()
		appendLog("开始下载： " + url)

		go func() {
			// resolve outputDir to absolute path (relative paths are relative to exeDir)
			actualOut := outputDir
			if !filepath.IsAbs(actualOut) {
				actualOut = filepath.Join(exeDir, actualOut)
			}
			// pass -o to yt-dlp to set output directory and filename template
			outTemplate := filepath.Join(actualOut, "%(title)s.%(ext)s")
			appendLog("使用输出目录: " + actualOut)
			cmd := exec.Command(exePath, "-o", outTemplate, url)
			cmd.Dir = filepath.Dir(exePath)

			// hide console window on Windows (affects child process window)
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
				appendLog("获取标准输出管道失败: " + err.Error())
				runningMu.Lock()
				running = false
				runningMu.Unlock()
				return
			}

			stderr, err := cmd.StderrPipe()
			if err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
				appendLog("获取错误输出管道失败: " + err.Error())
				runningMu.Lock()
				running = false
				runningMu.Unlock()
				return
			}

			if err := cmd.Start(); err != nil {
				// send a notification and log the error
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
				appendLog("启动失败: " + err.Error())
				runningMu.Lock()
				running = false
				runningMu.Unlock()
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			readPipe := func(r io.Reader) {
				defer wg.Done()
				br := bufio.NewReader(r)
				for {
					line, err := br.ReadBytes('\n')
					if len(line) > 0 {
						out := line
						// if bytes are not valid UTF-8 on Windows, try decode from GB18030/GBK
						if runtime.GOOS == "windows" && !utf8.Valid(line) {
							if dec, _, derr := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), line); derr == nil {
								out = dec
							} else if dec2, _, derr2 := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), line); derr2 == nil {
								out = dec2
							}
						}
						// trim trailing newline/carriage return
						text := strings.TrimRight(string(out), "\r\n")
						appendLog(text)
						// no percent parsing
					}
					if err != nil {
						if err == io.EOF {
							break
						}
						appendLog("读取子进程输出出错: " + err.Error())
						break
					}
				}
			}

			go readPipe(stdout)
			go readPipe(stderr)

			wg.Wait()

			err = cmd.Wait()
			if err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
				appendLog("下载失败: " + err.Error())
			} else {
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "完成", Content: "下载完成"})
				appendLog("下载完成")
			}
			runningMu.Lock()
			running = false
			runningMu.Unlock()
		}()
	}

	// decide initial button behavior based on availability of yt-dlp
	if exePath, ok := findYtDlp(); ok {
		downloadBtn.SetText("开始下载")
		downloadBtn.OnTapped = func() { startDownload(exePath) }
		// wire update button
		updateBtn.OnTapped = func() {
			appendLog("正在更新 yt-dlp: " + exePath)
			go func() {
				out, err := UpdateYtDlp(exePath, downloadProxy)
				if err != nil {
					appendLog("更新失败: " + err.Error())
					appendLog(out)
					fyne.Do(func() {
						dialog.ShowError(err, w)
					})
					return
				}
				appendLog("更新完成")
				appendLog(out)
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已更新"})
			}()
		}
	} else {
		updateBtn.Disable()
		downloadBtn.SetText("下载 yt-dlp")
		downloadBtn.OnTapped = func() {
			// download into exeDir
			appendLog("正在下载 yt-dlp 到: " + exeDir)
			appendLog("")
			go func() {
				// progress callback: log percent when possible, otherwise bytes
				var lastProgress string
				var lastPct int = -1
				onProgress := func(received, total int64) {
					var progress string
					var pct int
					if total > 0 {
						pct = int(float64(received) * 100.0 / float64(total))
						progress = fmt.Sprintf("下载 yt-dlp: %d%% (%d/%d)", pct, received, total)
					} else {
						// log every ~64KB
						progress = fmt.Sprintf("下载 yt-dlp: %d bytes", received)
					}

					// Only update if progress message has changed
					if (pct != lastPct) && (progress != lastProgress) {
						lastPct = pct
						lastProgress = progress
						fyne.Do(func() {
							rewriteLog(progress)
						})
					}
				}

				p, derr := DownloadYtDlpWithProgress(exeDir, downloadProxy, ytDlpURL, onProgress)
				if derr != nil {
					fyne.Do(func() {
						appendLog("下载 yt-dlp 失败: " + derr.Error())
						dialog.ShowError(derr, w)
					})
					return
				}
				fyne.Do(func() {
					appendLog("已下载: " + p)
					// after download, switch button to start download behavior and enable update
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已下载"})
					downloadBtn.SetText("开始下载")
					downloadBtn.OnTapped = func() { startDownload(p) }
					updateBtn.Enable()
					updateBtn.OnTapped = func() {
						appendLog("正在更新 yt-dlp: " + p)
						go func() {
							out, err := UpdateYtDlp(p, downloadProxy)
							if err != nil {
								appendLog("更新失败: " + err.Error())
								appendLog(out)
								fyne.Do(func() {
									dialog.ShowError(err, w)
								})
								return
							}
							appendLog("更新完成")
							appendLog(out)
							fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已更新"})
						}()
					}
				})
			}()
		}
	}

	// layout: top area contains an entry row (entry expands, clear button fixed at right)
	// use Border layout so entry fills remaining width and clearBtn stays on the right
	entryRow := container.NewBorder(nil, nil, nil, clearBtn, entry)
	buttons := container.NewHBox(setOutputBtn, widget.NewLabel("下载目录:"), linkContainer, layout.NewSpacer(), updateBtn, downloadBtn, aboutBtn)
	top := container.NewVBox(entryRow, buttons)
	content := container.NewBorder(top, nil, nil, nil, logScroll)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}
