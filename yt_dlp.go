package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"yinr.cc/yt-dlp-simpgo/utils"
)

// DownloadYtDlpWithProgress 下载 yt-dlp 并定期调用 onProgress(received, total)。
// total 为 -1 时表示未知。downloadProxy 非空时用作 HTTP(S) 代理。
// ytDlpURL 非空时替换默认的 GitHub 下载地址。
func DownloadYtDlpWithProgress(destDir string, downloadProxy string, ytDlpURL string, onProgress func(received, total int64)) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建目录: %w", err)
	}
	exeName := "yt-dlp"
	urlStr := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	if runtime.GOOS == "windows" {
		exeName = "yt-dlp.exe"
		urlStr = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	}
	outPath := filepath.Join(destDir, exeName)

	if ytDlpURL != "" {
		urlStr = ytDlpURL
	}

	client := &http.Client{}
	if downloadProxy != "" {
		proxyURL, perr := url.Parse(downloadProxy)
		if perr == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}

	resp, err := client.Get(urlStr)
	if err != nil {
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载失败: %s", resp.Status)
	}

	total := resp.ContentLength
	tmp := outPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("无法创建临时文件: %w", err)
	}

	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			wn, werr := f.Write(buf[:n])
			if werr != nil {
				_ = f.Close()
				_ = os.Remove(tmp)
				return "", fmt.Errorf("写入文件失败: %w", werr)
			}
			written += int64(wn)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			_ = f.Close()
			_ = os.Remove(tmp)
			return "", fmt.Errorf("读取响应失败: %w", rerr)
		}
	}

	// Windows 上必须先关闭文件再重命名，否则会报"文件被占用"
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("关闭文件失败: %w", err)
	}

	// Windows 上目标文件可能被其他进程锁定（杀毒扫描等），多次重试
	var renameErr error
	for i := 0; i < 6; i++ {
		renameErr = os.Rename(tmp, outPath)
		if renameErr == nil {
			break
		}
		if _, statErr := os.Stat(outPath); statErr == nil {
			_ = os.Remove(outPath)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if renameErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("重命名文件失败: %w", renameErr)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(outPath, 0755); err != nil {
			return outPath, fmt.Errorf("设置文件权限失败: %w", err)
		}
	}
	return outPath, nil
}

// UpdateYtDlp 运行 yt-dlp --update 并返回输出。
// downloadProxy 非空时注入 HTTP_PROXY/HTTPS_PROXY 环境变量。
func UpdateYtDlp(exePath string, downloadProxy string) (string, error) {
	cmd := utils.ExecCmd(exePath, "--update")

	if downloadProxy != "" {
		env := os.Environ()
		env = append(env, "HTTP_PROXY="+downloadProxy)
		env = append(env, "HTTPS_PROXY="+downloadProxy)
		env = append(env, "http_proxy="+downloadProxy)
		env = append(env, "https_proxy="+downloadProxy)
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("更新失败: %w", err)
	}
	return string(out), nil
}

// GetYtDlpVersion 获取本地 yt-dlp 版本。
func GetYtDlpVersion(exePath string) (string, error) {
	cmd := utils.ExecCmd(exePath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取版本失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GitHubRelease 用于解析 GitHub API 响应。
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// GetLatestYtDlpVersion 从 GitHub API 获取最新版本号。
func GetLatestYtDlpVersion(downloadProxy string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	if downloadProxy != "" {
		proxyURL, perr := url.Parse(downloadProxy)
		if perr == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}

	resp, err := client.Get("https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest")
	if err != nil {
		return "", fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API 返回错误: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return release.TagName, nil
}

// CheckYtDlpUpdate 检查是否有新版本，返回 (当前版本, 最新版本, 是否需要更新, 错误)。
func CheckYtDlpUpdate(exePath string, downloadProxy string) (current, latest string, needUpdate bool, err error) {
	current, err = GetYtDlpVersion(exePath)
	if err != nil {
		return "", "", false, err
	}

	latest, err = GetLatestYtDlpVersion(downloadProxy)
	if err != nil {
		return current, "", false, err
	}

	// 简单比较：去除可能的 v 前缀后比较
	currentClean := strings.TrimPrefix(current, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	needUpdate = currentClean != latestClean
	return current, latest, needUpdate, nil
}
