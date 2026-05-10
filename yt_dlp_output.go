package main

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var ytDlpDownloadProgressPattern = regexp.MustCompile(`^\[download\]\s+([0-9.]+)%\s+of\s+(.+?)(?:\s+at\s+(.+?))?(?:\s+ETA\s+(.+))?$`)

type YtDlpEvent struct {
	Text        string
	RewriteLast bool
	Progress    float64
	HasProgress bool
	Ok          bool
}

func friendlyYtDlpLine(line string) YtDlpEvent {
	if line == "" {
		return YtDlpEvent{}
	}

	lower := strings.ToLower(line)
	if strings.Contains(line, "ERROR:") || strings.Contains(lower, "error:") {
		return YtDlpEvent{Text: line, Ok: true}
	}
	if strings.Contains(line, "WARNING:") || strings.Contains(lower, "warning:") {
		return YtDlpEvent{Text: line, Ok: true}
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
		return YtDlpEvent{
			Text:        strings.Join(parts, "，"),
			RewriteLast: true,
			Progress:    progressValue / 100,
			HasProgress: true,
			Ok:          true,
		}
	}

	if strings.HasPrefix(line, "[download] Destination:") {
		return YtDlpEvent{Text: "保存文件: " + strings.TrimSpace(strings.TrimPrefix(line, "[download] Destination:")), Ok: true}
	}
	if strings.HasPrefix(line, "[download] ") && strings.Contains(line, "has already been downloaded") {
		return YtDlpEvent{Text: "文件已存在，跳过下载", Progress: 1, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[download] Downloading item ") {
		return YtDlpEvent{Text: strings.TrimPrefix(line, "[download] "), Ok: true}
	}
	if strings.HasPrefix(line, "[Merger]") {
		return YtDlpEvent{Text: "正在合并音视频", Progress: 0.95, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[ExtractAudio]") {
		return YtDlpEvent{Text: "正在提取音频", Progress: 0.95, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[EmbedSubtitle]") || strings.HasPrefix(line, "[EmbedThumbnail]") || strings.HasPrefix(line, "[Metadata]") {
		return YtDlpEvent{Text: "正在写入附加信息", Progress: 0.97, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[Fixup") {
		return YtDlpEvent{Text: "正在修复媒体文件", Progress: 0.98, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[info]") {
		return YtDlpEvent{Text: "已获取视频信息", Progress: 0.05, HasProgress: true, Ok: true}
	}
	if strings.HasPrefix(line, "[") && strings.Contains(line, "Downloading") {
		return YtDlpEvent{Text: "正在解析视频信息", RewriteLast: true, Progress: 0.02, HasProgress: true, Ok: true}
	}

	return YtDlpEvent{}
}

func readYtDlpPipe(r io.Reader, reporter ProgressReporter) {
	readPipe(r, func(line string) {
		ev := friendlyYtDlpLine(line)
		if !ev.Ok {
			return
		}
		if ev.HasProgress {
			reporter.setProgress(ev.Text, ev.Progress)
		}
		if ev.RewriteLast {
			reporter.rewriteLog(ev.Text)
		} else {
			reporter.appendLog(ev.Text)
		}
	})
}
