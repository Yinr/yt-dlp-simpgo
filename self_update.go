package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"yinr.cc/yt-dlp-simpgo/utils"
)

const appLatestReleaseAPI = "https://api.github.com/repos/Yinr/yt-dlp-simpgo/releases/latest"

var semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

type AppUpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	AssetName      string
	AssetURL       string
	Digest         string
	ReleaseURL     string
}

type appRelease struct {
	TagName string            `json:"tag_name"`
	HTMLURL string            `json:"html_url"`
	Assets  []appReleaseAsset `json:"assets"`
}

type appReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

func appHTTPClient(proxy string, timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
	return client
}

func currentPlatformAssetName() string {
	name := "yt-dlp-simpgo-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func parseSemver(version string) ([3]int, bool) {
	var parts [3]int
	matches := semverPattern.FindStringSubmatch(strings.TrimSpace(version))
	if matches == nil {
		return parts, false
	}
	for i := 0; i < 3; i++ {
		value, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return parts, false
		}
		parts[i] = value
	}
	return parts, true
}

func compareSemver(a, b string) (int, bool) {
	av, ok := parseSemver(a)
	if !ok {
		return 0, false
	}
	bv, ok := parseSemver(b)
	if !ok {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1, true
		}
		if av[i] > bv[i] {
			return 1, true
		}
	}
	return 0, true
}

func CheckAppUpdate(currentVersion string, proxy string) (*AppUpdateInfo, error) {
	if _, ok := parseSemver(currentVersion); !ok {
		return nil, nil
	}

	client := appHTTPClient(proxy, 30*time.Second)
	req, err := http.NewRequest(http.MethodGet, appLatestReleaseAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("创建更新检查请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "yt-dlp-simpgo/"+currentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("检查程序更新失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回错误: %s", resp.Status)
	}

	var release appRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析程序更新响应失败: %w", err)
	}

	cmp, ok := compareSemver(currentVersion, release.TagName)
	if !ok || cmp >= 0 {
		return nil, nil
	}

	assetName := currentPlatformAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName && asset.BrowserDownloadURL != "" {
			return &AppUpdateInfo{
				CurrentVersion: currentVersion,
				LatestVersion:  release.TagName,
				AssetName:      asset.Name,
				AssetURL:       asset.BrowserDownloadURL,
				Digest:         asset.Digest,
				ReleaseURL:     release.HTMLURL,
			}, nil
		}
	}

	return nil, fmt.Errorf("未找到当前平台更新文件: %s", assetName)
}

func DownloadAppUpdate(info *AppUpdateInfo, proxy string, onProgress func(received, total int64)) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法定位当前程序: %w", err)
	}

	ext := filepath.Ext(info.AssetName)
	tmp, err := os.CreateTemp(filepath.Dir(exePath), ".yt-dlp-simpgo-update-*"+ext)
	if err != nil {
		return "", fmt.Errorf("无法创建临时更新文件: %w", err)
	}
	tmpPath := tmp.Name()
	defer tmp.Close()

	client := appHTTPClient(proxy, 10*time.Minute)
	req, err := http.NewRequest(http.MethodGet, info.AssetURL, nil)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("创建更新下载请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "yt-dlp-simpgo/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("下载程序更新失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("下载程序更新失败: %s", resp.Status)
	}

	hash := sha256.New()
	if err := copyWithProgress(io.MultiWriter(tmp, hash), resp.Body, resp.ContentLength, onProgress); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("写入程序更新失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("关闭程序更新文件失败: %w", err)
	}

	if err := verifyAssetDigest(info.Digest, hash.Sum(nil)); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			_ = os.Remove(tmpPath)
			return "", fmt.Errorf("设置程序更新文件权限失败: %w", err)
		}
	}

	return tmpPath, nil
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64, onProgress func(received, total int64)) error {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			wn, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			written += int64(wn)
			if onProgress != nil {
				onProgress(written, total)
			}
			if wn != n {
				return io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}
	return nil
}

func verifyAssetDigest(digest string, sum []byte) error {
	if digest == "" {
		return nil
	}
	const prefix = "sha256:"
	if !strings.HasPrefix(strings.ToLower(digest), prefix) {
		return nil
	}
	expected := strings.TrimPrefix(strings.ToLower(digest), prefix)
	actual := hex.EncodeToString(sum)
	if expected != actual {
		return fmt.Errorf("程序更新校验失败: sha256 不匹配")
	}
	return nil
}

func InstallAppUpdateAndRestart(updatePath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法定位当前程序: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	if runtime.GOOS == "windows" {
		return installAppUpdateWindows(updatePath, exePath, exeDir)
	}

	if err := os.Rename(updatePath, exePath); err != nil {
		return fmt.Errorf("替换程序失败: %w", err)
	}
	cmd := exec.Command(exePath)
	cmd.Dir = exeDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("重启程序失败: %w", err)
	}
	return nil
}

func installAppUpdateWindows(updatePath string, exePath string, exeDir string) error {
	scriptPath := filepath.Join(exeDir, ".yt-dlp-simpgo-update.ps1")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$pidToWait = %d
$source = %s
$target = %s
$workdir = %s
Wait-Process -Id $pidToWait -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500
for ($i = 0; $i -lt 20; $i++) {
    try {
        Move-Item -LiteralPath $source -Destination $target -Force
        break
    } catch {
        if ($i -eq 19) { throw }
        Start-Sleep -Milliseconds 300
    }
}
Start-Process -FilePath $target -WorkingDirectory $workdir
Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
`, os.Getpid(), powershellQuote(updatePath), powershellQuote(exePath), powershellQuote(exeDir))

	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return fmt.Errorf("创建更新脚本失败: %w", err)
	}

	cmd := utils.ExecCmd("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	cmd.Dir = exeDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新脚本失败: %w", err)
	}
	return nil
}

func powershellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
