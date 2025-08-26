package main

import (
	"bufio"
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
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func main() {
	a := app.New()
	w := a.NewWindow("视频下载工具")
	w.Resize(fyne.NewSize(720, 420))

	// URL entry (single-line but taller for easier editing)
	entry := widget.NewEntry()
	entry.SetPlaceHolder("在此输入视频网址，例如：https://...")

	// output directory (default ./下载)
	outputDir := filepath.Join(".", "下载")
	_ = os.MkdirAll(outputDir, 0755)
	outputBinding := binding.NewString()
	_ = outputBinding.Set(outputDir)
	// create a small container to hold the clickable output link
	linkContainer := container.NewHBox()
	setOutputBtn := widget.NewButton("设置下载目录", func() {
		dialog.ShowFolderOpen(func(li fyne.ListableURI, err error) {
			if err != nil || li == nil {
				return
			}
			p := li.Path()
			if p == "" {
				return
			}
			outputDir = p
			_ = os.MkdirAll(outputDir, 0755)
			_ = outputBinding.Set(outputDir)
		}, w)
	})

	// helper to open folder (platform-specific)
	openFolder := func(path string, fileURL *url.URL) {
		if runtime.GOOS == "windows" {
			_ = exec.Command("explorer", path).Start()
		} else if fileURL != nil {
			_ = fyne.CurrentApp().OpenURL(fileURL)
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

	downloadBtn := widget.NewButton("开始下载", nil)

	// thread-safe running flag to prevent re-entrancy
	var runningMu sync.Mutex
	running := false

	// helper to append to log on main thread
	// add a listener so when binding changes we auto-scroll; binding listeners
	// are invoked on the UI thread so calling ScrollToBottom is safe here.
	logBinding.AddListener(binding.NewDataListener(func() {
		logScroll.ScrollToBottom()
	}))

	appendLog := func(text string) {
		// append via binding (get current, set new)
		go func() {
			cur, _ := logBinding.Get()
			if cur == "" {
				_ = logBinding.Set(text)
			} else {
				_ = logBinding.Set(cur + "\n" + text)
			}
		}()
	}

	// download action
	downloadBtn.OnTapped = func() {
		url := strings.TrimSpace(entry.Text)
		if url == "" {
			dialog.ShowInformation("提示", "请输入一个有效的网址", w)
			return
		}

		// find yt-dlp executable in current directory
		exeName := "yt-dlp"
		if runtime.GOOS == "windows" {
			exeName = "yt-dlp.exe"
		}
		exePath, _ := filepath.Abs(exeName)
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			dialog.ShowError(err, w)
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
			// pass -o to yt-dlp to set output directory and filename template
			outTemplate := filepath.Join(outputDir, "%(title)s.%(ext)s")
			appendLog("使用输出目录: " + outputDir)
			cmd := exec.Command(exePath, "-o", outTemplate, url)
			cmd.Dir = filepath.Dir(exePath)

			// hide console window on Windows (affects child process window)
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}

			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

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
							if dec, _, derr := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), line); derr == nil {
								out = dec
							} else if dec2, _, derr2 := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), line); derr2 == nil {
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

			err := cmd.Wait()
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

	// layout: top area contains an entry row (entry expands, clear button fixed at right)
	// use Border layout so entry fills remaining width and clearBtn stays on the right
	entryRow := container.NewBorder(nil, nil, nil, clearBtn, entry)
	buttons := container.NewHBox(setOutputBtn, widget.NewLabel("下载目录:"), linkContainer, layout.NewSpacer(), downloadBtn)
	top := container.NewVBox(entryRow, buttons)
	content := container.NewBorder(top, nil, nil, nil, logScroll)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}
