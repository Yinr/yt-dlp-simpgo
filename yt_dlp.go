package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// DownloadYtDlp downloads the yt-dlp binary into destDir and returns the saved path.
// DownloadYtDlpWithProgress downloads yt-dlp and calls onProgress(received, total)
// periodically. total may be -1 if unknown.
// DownloadYtDlpWithProgress downloads yt-dlp and calls onProgress(received, total)
// periodically. total may be -1 if unknown. If downloadProxy is non-empty it
// will be used as the HTTP(S) proxy. If ytDlpURL is non-empty it will be used
// instead of the default GitHub URL.
func DownloadYtDlpWithProgress(destDir string, downloadProxy string, ytDlpURL string, onProgress func(received, total int64)) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	exeName := "yt-dlp"
	urlStr := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	if runtime.GOOS == "windows" {
		exeName = "yt-dlp.exe"
		urlStr = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	}
	outPath := filepath.Join(destDir, exeName)

	// allow custom URL override
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
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	total := resp.ContentLength
	tmp := outPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// copy with progress reporting
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			wn, werr := f.Write(buf[:n])
			if werr != nil {
				_ = os.Remove(tmp)
				return "", werr
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
			_ = os.Remove(tmp)
			return "", rerr
		}
	}

	// make sure file is closed before renaming on Windows (prevents "file in use")
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	// Try to rename tmp -> outPath, with retries. On Windows this can fail
	// if the target exists and is locked by another process (e.g. antivirus
	// scanning or running executable). Retry a few times and attempt to
	// remove the existing target if present between attempts.
	var renameErr error
	for range 6 {
		renameErr = os.Rename(tmp, outPath)
		if renameErr == nil {
			break
		}
		// if target exists, try to remove it and retry
		if _, statErr := os.Stat(outPath); statErr == nil {
			_ = os.Remove(outPath)
			// small pause before retrying
			time.Sleep(200 * time.Millisecond)
			continue
		}
		// otherwise pause and retry
		time.Sleep(200 * time.Millisecond)
	}
	if renameErr != nil {
		// final cleanup: try to remove tmp to avoid leaving partial file
		_ = os.Remove(tmp)
		return "", renameErr
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(outPath, 0755); err != nil {
			return outPath, err
		}
	}
	return outPath, nil
}

// Compatibility wrapper without progress callback
func DownloadYtDlp(destDir string) (string, error) {
	return DownloadYtDlpWithProgress(destDir, "", "", nil)
}

// UpdateYtDlp runs the existing yt-dlp executable with --update and returns its combined output.
// If downloadProxy is provided it will be injected into the command environment as HTTP_PROXY/HTTPS_PROXY.
func UpdateYtDlp(exePath string, downloadProxy string) (string, error) {
	cmd := exec.Command(exePath, "--update")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if downloadProxy != "" {
		env := os.Environ()
		env = append(env, "HTTP_PROXY="+downloadProxy)
		env = append(env, "HTTPS_PROXY="+downloadProxy)
		env = append(env, "http_proxy="+downloadProxy)
		env = append(env, "https_proxy="+downloadProxy)
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
