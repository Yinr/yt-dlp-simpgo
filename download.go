package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"yinr.cc/yt-dlp-simpgo/utils"
)

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

func wireUpdateBtn(btn *widget.Button, exePath string, downloadProxy string, reporter ProgressReporter) {
	btn.Enable()
	btn.OnTapped = func() {
		reporter.appendLog("正在更新 yt-dlp: " + exePath)
		reporter.setProgress("正在更新 yt-dlp", -1)
		go func() {
			out, err := UpdateYtDlp(exePath, downloadProxy)
			if err != nil {
				reporter.appendLog(logMarker("yt-dlp 更新失败"))
				reporter.appendLog("更新失败: " + err.Error())
				reporter.appendLog(out)
				reporter.clear()
				fyne.Do(func() {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				})
				return
			}
			reporter.setProgress("yt-dlp 更新完成", 1)
			reporter.appendLog(logMarker("yt-dlp 更新完成"))
			reporter.appendLog("更新完成")
			reporter.appendLog(out)
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "已完成", Content: "yt-dlp 已更新"})
		}()
	}
}

func startDownload(exePath string, url string, outputDir string, exeDir string,
	runningMu *sync.Mutex, running *bool, reporter ProgressReporter) {

	runningMu.Lock()
	if *running {
		runningMu.Unlock()
		dialog.ShowInformation("提示", "已有下载在进行中", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	*running = true
	runningMu.Unlock()
	reporter.appendLog(logMarker("开始下载"))
	reporter.appendLog("下载地址: " + url)
	reporter.setProgress("准备下载", 0)

	go func() {
		actualOut := outputDir
		if !filepath.IsAbs(actualOut) {
			actualOut = filepath.Join(exeDir, actualOut)
		}
		outTemplate := filepath.Join(actualOut, "%(title)s.%(ext)s")
		confPath := filepath.Join(exeDir, YTDLPConfName)
		reporter.appendLog("使用输出目录: " + actualOut)
		reporter.appendLog("使用配置文件: " + confPath)
		cmd := utils.ExecCmd(exePath, "--newline", "--no-colors", "--config-location", confPath, "-o", outTemplate, url)
		cmd.Dir = exeDir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			reporter.appendLog("获取标准输出管道失败: " + err.Error())
			reporter.clear()
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			reporter.appendLog("获取错误输出管道失败: " + err.Error())
			reporter.clear()
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		if err := cmd.Start(); err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			reporter.appendLog("启动失败: " + err.Error())
			reporter.clear()
			runningMu.Lock()
			*running = false
			runningMu.Unlock()
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); readYtDlpPipe(stdout, reporter) }()
		go func() { defer wg.Done(); readYtDlpPipe(stderr, reporter) }()
		wg.Wait()

		err = cmd.Wait()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "下载失败", Content: err.Error()})
			reporter.appendLog(logMarker("下载失败"))
			reporter.appendLog("下载失败: " + err.Error())
			reporter.clear()
		} else {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "完成", Content: "下载完成"})
			reporter.setProgress("下载完成", 1)
			reporter.appendLog(logMarker("下载完成"))
			reporter.appendLog("下载完成")
		}
		runningMu.Lock()
		*running = false
		runningMu.Unlock()
	}()
}
