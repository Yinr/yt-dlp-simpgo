package main

import (
	"fmt"
	"os"

	_ "embed"

	ini "gopkg.in/ini.v1"
)

// Configuration is now stored in two files:
// - yt-dlp-simpgo.ini : only program settings (section [app], key output_dir)
// - yt-dlp.conf       : the yt-dlp configuration (full file, not embedded into ini)
// Defaults are embedded in default_conf.go as defaultYTDLPConf and defaultIniConf.

// filenames used by the application (centralized to avoid duplication)
const (
	IniFileName   = "yt-dlp-simpgo.ini"
	YTDLPConfName = "yt-dlp.conf"
)

// embedded defaults (pulled from res/ at build time)
//
//go:embed res/yt-dlp.conf
var defaultYTDLPConf string

//go:embed res/yt-dlp-simpgo.ini
var defaultIniConf string

// LoadConfig loads program settings from the ini file.
// It returns outputDir, downloadProxy, ytDlpURL and an error if occurred.
func LoadConfig(path string) (outputDir string, downloadProxy string, ytDlpURL string, err error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return "", "", "", nil
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return "", "", "", fmt.Errorf("无法加载配置文件: %w", err)
	}
	sec := cfg.Section("app")
	outputDir = sec.Key("output_dir").MustString("")
	downloadProxy = sec.Key("download_proxy").MustString("")
	ytDlpURL = sec.Key("yt_dlp_url").MustString("")
	return outputDir, downloadProxy, ytDlpURL, nil
}

// SaveConfig writes only the program settings (currently output_dir) into the ini file.
// SaveConfig writes program settings (output_dir, download_proxy, yt_dlp_url) into the ini file.
func SaveConfig(path, outputDir, downloadProxy, ytDlpURL string) error {
	cfg := ini.Empty()
	app := cfg.Section("app")
	app.Key("output_dir").SetValue(outputDir)
	app.Key("download_proxy").SetValue(downloadProxy)
	app.Key("yt_dlp_url").SetValue(ytDlpURL)
	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("无法保存配置文件: %w", err)
	}
	return nil
}

// EnsureDefaults ensures yt-dlp.conf and the ini file exist. It will:
//   - create yt-dlp.conf from embedded defaultYTDLPConf if the file is missing;
//   - create the ini file from embedded defaultIniConf if missing, or write a minimal
//     ini containing output_dir when no embedded ini is available.
//
// It returns the resolved outputDir (either the provided defaultOutputDir or the
// value read from an existing ini) and any error encountered.
func EnsureDefaults(iniPath, defaultOutputDir string) (outputDir string, downloadProxy string, ytDlpURL string, err error) {
	outputDir = defaultOutputDir

	// If yt-dlp.conf doesn't exist, write embedded default
	if _, st := os.Stat(YTDLPConfName); os.IsNotExist(st) {
		if defaultYTDLPConf != "" {
			if werr := writeUTF8BOMFile(YTDLPConfName, defaultYTDLPConf, 0644); werr != nil {
				return outputDir, "", "", fmt.Errorf("无法写入默认yt-dlp配置: %w", werr)
			}
		}
	}

	// If ini exists, read output_dir and other settings from it
	if _, serr := os.Stat(iniPath); serr == nil {
		if od, dp, yurl, rerr := LoadConfig(iniPath); rerr == nil {
			if od != "" {
				outputDir = od
			}
			downloadProxy = dp
			ytDlpURL = yurl
		} else {
			// return read error
			return outputDir, "", "", fmt.Errorf("无法加载配置: %w", rerr)
		}
	} else {
		// ini missing: if we have an embedded ini template, write it; otherwise write minimal ini
		if defaultIniConf != "" {
			if werr := os.WriteFile(iniPath, []byte(defaultIniConf), 0644); werr != nil {
				return outputDir, "", "", fmt.Errorf("无法写入默认配置: %w", werr)
			}
			// after writing embedded ini, try to read settings
			if od, dp, yurl, rerr := LoadConfig(iniPath); rerr == nil {
				if od != "" {
					outputDir = od
				}
				downloadProxy = dp
				ytDlpURL = yurl
			} else {
				return outputDir, "", "", fmt.Errorf("无法读取新创建的配置: %w", rerr)
			}
		} else {
			if werr := SaveConfig(iniPath, outputDir, "", ""); werr != nil {
				return outputDir, "", "", fmt.Errorf("无法保存配置: %w", werr)
			}
		}
	}

	// ensure output directory exists
	if mkerr := os.MkdirAll(outputDir, 0755); mkerr != nil {
		return outputDir, downloadProxy, ytDlpURL, fmt.Errorf("无法创建输出目录 %s: %w", outputDir, mkerr)
	}

	return outputDir, downloadProxy, ytDlpURL, nil
}

// writeUTF8BOMFile writes a string to path with a UTF-8 BOM prefix.
func writeUTF8BOMFile(path, content string, perm os.FileMode) error {
	bom := []byte{0xEF, 0xBB, 0xBF}
	data := append(bom, []byte(content)...)
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("无法写入文件 %s: %w", path, err)
	}
	return nil
}