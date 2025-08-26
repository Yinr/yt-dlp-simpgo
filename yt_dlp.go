package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// DownloadYtDlp downloads the yt-dlp binary into destDir and returns the saved path.
// DownloadYtDlpWithProgress downloads yt-dlp and calls onProgress(received, total)
// periodically. total may be -1 if unknown.
func DownloadYtDlpWithProgress(destDir string, onProgress func(received, total int64)) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	exeName := "yt-dlp"
	url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	if runtime.GOOS == "windows" {
		exeName = "yt-dlp.exe"
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	}
	outPath := filepath.Join(destDir, exeName)

	resp, err := http.Get(url)
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

	if err := os.Rename(tmp, outPath); err != nil {
		return "", err
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
	return DownloadYtDlpWithProgress(destDir, nil)
}

// UpdateYtDlp runs the existing yt-dlp executable with --update and returns its combined output.
func UpdateYtDlp(exePath string) (string, error) {
	cmd := exec.Command(exePath, "--update")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
