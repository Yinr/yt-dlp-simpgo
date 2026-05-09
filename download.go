package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"yinr.cc/yt-dlp-simpgo/utils"
)

// findYtDlp checks for yt-dlp in exeDir first, then PATH.
func findYtDlp(exeDir string) (string, bool) {
	exeName := "yt-dlp"
	if runtime.GOOS == "windows" {
		exeName = "yt-dlp.exe"
	}
	p := filepath.Join(exeDir, exeName)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	if pathP, err := exec.LookPath(exeName); err == nil {
		return pathP, true
	}
	return "", false
}

// readPipe reads lines from r, decodes GBK on Windows if needed, and calls appendLog.
func readPipe(r io.Reader, appendLog func(string)) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			out := line
			if runtime.GOOS == "windows" && !utf8.Valid(line) {
				if dec, _, derr := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), line); derr == nil {
					out = dec
				} else if dec2, _, derr2 := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), line); derr2 == nil {
					out = dec2
				}
			}
			text := strings.TrimRight(string(out), "\r\n")
			appendLog(text)
		}
		if err != nil {
			if err != io.EOF {
				appendLog("读取子进程输出出错: " + err.Error())
			}
			break
		}
	}
}

// wireUpdateBtn sets up the update button to run yt-dlp --update.
func wireUpdateBtn(btn *widget.Button, exePath string, downloadProxy string, appendLog func(string)) {
	btn.Enable()
	btn.OnTapped = func() {
		appendLog("正在更新 yt-dlp: " + exePath)
		go func() {
			out, err := UpdateYtDlp(exePath, downloadProxy)
			if err != nil {
				appendLog("更新失败: " + err.Error())
				appendLog(out)
				fyne.Do(func() {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				})
				return
			}
			appendLog("更新完成")
			appendLog(out)
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已更新"})
		}()
	}
}

// startDownload launches a yt-dlp download in a goroutine.
func startDownload(exePath string, url string, outputDir string, exeDir string,
	runningMu *sync.Mutex, running *bool, appendLog func(string)) {

	runningMu.Lock()
	if *running {
		runningMu.Unlock()
		dialog.ShowInformation("提示", "已有下载在进行中", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	*running = true
	runningMu.Unlock()
	appendLog("开始下载： " + url)

	go func() {
		actualOut := outputDir
		if !filepath.IsAbs(actualOut) {
			actualOut = filepath.Join(exeDir, actualOut)
		}
		outTemplate := filepath.Join(actualOut, "%(title)s.%(ext)s")
		confPath := filepath.Join(exeDir, YTDLPConfName)
		appendLog("使用输出目录: " + actualOut)
		appendLog("使用配置文件: " + confPath)
		cmd := utils.ExecCmd(exePath, "--config-location", confPath, "-o", outTemplate, url)
		cmd.Dir = exeDir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			appendLog("获取标准输出管道失败: " + err.Error())
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			appendLog("获取错误输出管道失败: " + err.Error())
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		if err := cmd.Start(); err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			appendLog("启动失败: " + err.Error())
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); readPipe(stdout, appendLog) }()
		go func() { defer wg.Done(); readPipe(stderr, appendLog) }()
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
		*running = false
		runningMu.Unlock()
	}()
}
