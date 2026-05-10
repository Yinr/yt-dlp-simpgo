package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

type ProgressReporter struct {
	AppendLog   func(string)
	RewriteLog  func(string)
	SetProgress func(status string, value float64)
	Clear       func()
}

func (p ProgressReporter) appendLog(text string) {
	if p.AppendLog != nil {
		p.AppendLog(text)
	}
}

func (p ProgressReporter) rewriteLog(text string) {
	if p.RewriteLog != nil {
		p.RewriteLog(text)
	} else {
		p.appendLog(text)
	}
}

func (p ProgressReporter) setProgress(status string, value float64) {
	if p.SetProgress != nil {
		p.SetProgress(status, value)
	}
}

func (p ProgressReporter) clear() {
	if p.Clear != nil {
		p.Clear()
	}
}

var ytDlpDownloadProgressPattern = regexp.MustCompile(`^\[download\]\s+([0-9.]+)%\s+of\s+(.+?)(?:\s+at\s+(.+?))?(?:\s+ETA\s+(.+))?$`)

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

func decodeProcessLine(line []byte) string {
	out := line
	if runtime.GOOS == "windows" && !utf8.Valid(line) {
		if dec, _, derr := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), line); derr == nil {
			out = dec
		} else if dec2, _, derr2 := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), line); derr2 == nil {
			out = dec2
		}
	}
	return strings.TrimSpace(strings.TrimRight(string(out), "\r\n"))
}

func friendlyYtDlpLine(line string) (text string, rewriteLast bool, progress float64, ok bool) {
	if line == "" {
		return "", false, 0, false
	}

	lower := strings.ToLower(line)
	if strings.Contains(line, "ERROR:") || strings.Contains(lower, "error:") {
		return line, false, 0, true
	}
	if strings.Contains(line, "WARNING:") || strings.Contains(lower, "warning:") {
		return line, false, 0, true
	}

	if matches := ytDlpDownloadProgressPattern.FindStringSubmatch(line); matches != nil {
		percent := strings.TrimSpace(matches[1])
		total := strings.TrimSpace(matches[2])
		speed := strings.TrimSpace(matches[3])
		eta := strings.TrimSpace(matches[4])

		parts := []string{fmt.Sprintf("下载中: %s%%", percent)}
		if total != "" {
			parts = append(parts, "总大小 "+total)
		}
		if speed != "" {
			parts = append(parts, "速度 "+speed)
		}
		if eta != "" {
			parts = append(parts, "剩余 "+eta)
		}
		progressValue, _ := strconv.ParseFloat(percent, 64)
		return strings.Join(parts, "，"), true, progressValue / 100, true
	}

	if strings.HasPrefix(line, "[download] Destination:") {
		return "保存文件: " + strings.TrimSpace(strings.TrimPrefix(line, "[download] Destination:")), false, 0, true
	}
	if strings.HasPrefix(line, "[download] ") && strings.Contains(line, "has already been downloaded") {
		return "文件已存在，跳过下载", false, 1, true
	}
	if strings.HasPrefix(line, "[download] Downloading item ") {
		return strings.TrimPrefix(line, "[download] "), false, 0, true
	}
	if strings.HasPrefix(line, "[Merger]") {
		return "正在合并音视频", false, 0.95, true
	}
	if strings.HasPrefix(line, "[ExtractAudio]") {
		return "正在提取音频", false, 0.95, true
	}
	if strings.HasPrefix(line, "[EmbedSubtitle]") || strings.HasPrefix(line, "[EmbedThumbnail]") || strings.HasPrefix(line, "[Metadata]") {
		return "正在写入附加信息", false, 0.97, true
	}
	if strings.HasPrefix(line, "[Fixup") {
		return "正在修复媒体文件", false, 0.98, true
	}
	if strings.HasPrefix(line, "[info]") {
		return "已获取视频信息", false, 0.05, true
	}
	if strings.HasPrefix(line, "[") && strings.Contains(line, "Downloading") {
		return "正在解析视频信息", true, 0.02, true
	}

	return "", false, 0, false
}

// readPipe reads lines from r, decodes GBK on Windows if needed, and calls appendLog.
func readPipe(r io.Reader, appendLog func(string)) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			appendLog(decodeProcessLine(line))
		}
		if err != nil {
			if err != io.EOF {
				appendLog("读取子进程输出出错: " + err.Error())
			}
			break
		}
	}
}

func readYtDlpPipe(r io.Reader, reporter ProgressReporter) {
	readPipe(r, func(line string) {
		text, rewriteLast, progress, ok := friendlyYtDlpLine(line)
		if !ok {
			return
		}
		if progress >= 0 {
			reporter.setProgress(text, progress)
		}
		if rewriteLast {
			reporter.rewriteLog(text)
		} else {
			reporter.appendLog(text)
		}
	})
}

// wireUpdateBtn sets up the update button to run yt-dlp --update.
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

// startDownload launches a yt-dlp download in a goroutine.
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
