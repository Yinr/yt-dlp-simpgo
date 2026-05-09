package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	_ "embed"

	ini "gopkg.in/ini.v1"
)

// 配置分为两个文件：
// - yt-dlp-simpgo.ini : 程序设置（section [app]）
// - yt-dlp.conf       : yt-dlp 配置
// 默认值通过 go:embed 从 res/ 目录嵌入二进制文件。

const (
	IniFileName   = "yt-dlp-simpgo.ini"
	YTDLPConfName = "yt-dlp.conf"
)

// AppConfig holds program-level settings.
type AppConfig struct {
	OutputDir     string
	DownloadProxy string
	YtDlpURL      string
}

// YTDLPConfig holds yt-dlp command-line options.
type YTDLPConfig struct {
	UseProxy          bool
	ProxyAddress      string
	WriteSubs         bool
	SubLangs          string
	EmbedSubs         bool
	EmbedChapters     bool
	SplitChapters     bool
	EmbedMetadata     bool
	MergeOutputFormat string // "mp4/mkv", "mp4", "mkv", "auto"
	ExtraArgs         string
}

// DefaultYTDLPConfig returns a YTDLPConfig with sensible defaults.
func DefaultYTDLPConfig() *YTDLPConfig {
	return &YTDLPConfig{
		WriteSubs:         true,
		SubLangs:          "chs,cht,zh.*",
		EmbedSubs:         true,
		EmbedChapters:     true,
		EmbedMetadata:     true,
		MergeOutputFormat: "mp4/mkv",
	}
}

// embedded defaults (pulled from res/ at build time)
//
//go:embed res/yt-dlp.conf
var defaultYTDLPConf string

//go:embed res/yt-dlp-simpgo.ini
var defaultIniConf string

//go:embed res/icon.png
var iconData []byte

// LoadAppConfig loads program settings from the ini file.
func LoadAppConfig(path string) (*AppConfig, error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return &AppConfig{}, nil
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("无法加载配置文件: %w", err)
	}
	sec := cfg.Section("app")
	return &AppConfig{
		OutputDir:     sec.Key("output_dir").MustString(""),
		DownloadProxy: sec.Key("download_proxy").MustString(""),
		YtDlpURL:      sec.Key("yt_dlp_url").MustString(""),
	}, nil
}

// SaveAppConfig writes program settings into the ini file.
func SaveAppConfig(path string, cfg *AppConfig) error {
	iniCfg := ini.Empty()
	app := iniCfg.Section("app")
	app.Key("output_dir").SetValue(cfg.OutputDir)
	app.Key("download_proxy").SetValue(cfg.DownloadProxy)
	app.Key("yt_dlp_url").SetValue(cfg.YtDlpURL)
	if err := iniCfg.SaveTo(path); err != nil {
		return fmt.Errorf("无法保存配置文件: %w", err)
	}
	return nil
}

// LoadYTDLPConf loads yt-dlp configuration from file.
func LoadYTDLPConf(path string) (*YTDLPConfig, error) {
	cfg := DefaultYTDLPConfig()

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var extraLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 处理注释形式的禁用选项（如 # --embed-chapters）
		if strings.HasPrefix(line, "#") {
			commentLine := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			commentParts := strings.Fields(commentLine)
			if len(commentParts) > 0 {
				switch commentParts[0] {
				case "--write-subs":
					cfg.WriteSubs = false
				case "--embed-subs":
					cfg.EmbedSubs = false
				case "--embed-chapters":
					cfg.EmbedChapters = false
				case "--split-chapters":
					cfg.SplitChapters = false
				case "--embed-metadata":
					cfg.EmbedMetadata = false
				case "--merge-output-format":
					// 注释形式的格式选项不处理，保持默认
				default:
					extraLines = append(extraLines, scanner.Text())
				}
			} else {
				extraLines = append(extraLines, scanner.Text())
			}
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			extraLines = append(extraLines, scanner.Text())
			continue
		}

		switch parts[0] {
		case "--proxy":
			if len(parts) > 1 {
				cfg.UseProxy = true
				cfg.ProxyAddress = parts[1]
			} else {
				cfg.UseProxy = false
			}
		case "--write-subs":
			cfg.WriteSubs = true
		case "--no-write-subs":
			cfg.WriteSubs = false
		case "--sub-langs":
			if len(parts) > 1 {
				cfg.SubLangs = strings.Trim(parts[1], "\"'")
			}
		case "--embed-subs":
			cfg.EmbedSubs = true
		case "--no-embed-subs":
			cfg.EmbedSubs = false
		case "--embed-chapters":
			cfg.EmbedChapters = true
		case "--no-embed-chapters":
			cfg.EmbedChapters = false
		case "--split-chapters":
			cfg.SplitChapters = true
		case "--no-split-chapters":
			cfg.SplitChapters = false
		case "--embed-metadata":
			cfg.EmbedMetadata = true
		case "--no-embed-metadata":
			cfg.EmbedMetadata = false
		case "--merge-output-format":
			if len(parts) > 1 {
				cfg.MergeOutputFormat = strings.Trim(parts[1], "\"'")
			}
		default:
			extraLines = append(extraLines, scanner.Text())
		}
	}

	cfg.ExtraArgs = strings.Join(extraLines, "\n")
	return cfg, nil
}

// SaveYTDLPConf saves yt-dlp configuration to file.
func SaveYTDLPConf(path string, cfg *YTDLPConfig) error {
	var lines []string

	lines = append(lines, "")
	lines = append(lines, "# 使用网络代理")
	if cfg.UseProxy && cfg.ProxyAddress != "" {
		lines = append(lines, "--proxy "+cfg.ProxyAddress)
	} else {
		lines = append(lines, "# --proxy 127.0.0.1:7890")
	}

	lines = append(lines, "")
	lines = append(lines, "# 同时下载字幕")
	if cfg.WriteSubs {
		lines = append(lines, "--write-subs")
	} else {
		lines = append(lines, "# --write-subs")
	}
	lines = append(lines, "# 字幕语言")
	lines = append(lines, "--sub-langs \""+cfg.SubLangs+"\"")
	lines = append(lines, "# 内嵌字幕")
	if cfg.EmbedSubs {
		lines = append(lines, "--embed-subs")
	} else {
		lines = append(lines, "# --embed-subs")
	}

	lines = append(lines, "")
	lines = append(lines, "# 同时下载章节")

	lines = append(lines, "")
	lines = append(lines, "# 内嵌章节")
	if cfg.EmbedChapters {
		lines = append(lines, "--embed-chapters")
	} else {
		lines = append(lines, "# --embed-chapters")
	}

	lines = append(lines, "")
	lines = append(lines, "# 拆分章节")
	if cfg.SplitChapters {
		lines = append(lines, "--split-chapters")
	} else {
		lines = append(lines, "# --split-chapters")
	}

	lines = append(lines, "")
	lines = append(lines, "# 内嵌元数据")
	if cfg.EmbedMetadata {
		lines = append(lines, "--embed-metadata")
	} else {
		lines = append(lines, "# --embed-metadata")
	}

	lines = append(lines, "")
	lines = append(lines, "# 合并容器输出格式")
	if cfg.MergeOutputFormat != "auto" && cfg.MergeOutputFormat != "" {
		lines = append(lines, "--merge-output-format \""+cfg.MergeOutputFormat+"\"")
	} else {
		lines = append(lines, "# --merge-output-format \"mp4/mkv\"")
	}

	if cfg.ExtraArgs != "" {
		lines = append(lines, "")
		for _, line := range strings.Split(cfg.ExtraArgs, "\n") {
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n")
	return writeUTF8BOMFile(path, content, 0644)
}

// EnsureDefaults ensures yt-dlp.conf and the ini file exist. It will:
//   - create yt-dlp.conf from embedded defaultYTDLPConf if the file is missing;
//   - create the ini file from embedded defaultIniConf if missing, or write a minimal
//     ini containing output_dir when no embedded ini is available.
//
// It returns the loaded AppConfig, YTDLPConfig and any error encountered.
func EnsureDefaults(iniPath, defaultOutputDir string) (*AppConfig, *YTDLPConfig, error) {
	appCfg := &AppConfig{
		OutputDir: defaultOutputDir,
	}

	ytdlpCfg := DefaultYTDLPConfig()

	// If yt-dlp.conf doesn't exist, write embedded default
	if _, st := os.Stat(YTDLPConfName); os.IsNotExist(st) {
		if defaultYTDLPConf != "" {
			if werr := writeUTF8BOMFile(YTDLPConfName, defaultYTDLPConf, 0644); werr != nil {
				return nil, nil, fmt.Errorf("无法写入默认yt-dlp配置: %w", werr)
			}
		}
	}

	// Load yt-dlp.conf
	if loadedCfg, err := LoadYTDLPConf(YTDLPConfName); err == nil {
		ytdlpCfg = loadedCfg
	}

	// If ini exists, read settings from it
	if _, serr := os.Stat(iniPath); serr == nil {
		if loadedCfg, rerr := LoadAppConfig(iniPath); rerr == nil {
			if loadedCfg.OutputDir != "" {
				appCfg.OutputDir = loadedCfg.OutputDir
			}
			appCfg.DownloadProxy = loadedCfg.DownloadProxy
			appCfg.YtDlpURL = loadedCfg.YtDlpURL
		} else {
			return nil, nil, fmt.Errorf("无法加载配置: %w", rerr)
		}
	} else {
		// ini missing: if we have an embedded ini template, write it; otherwise write minimal ini
		if defaultIniConf != "" {
			if werr := os.WriteFile(iniPath, []byte(defaultIniConf), 0644); werr != nil {
				return nil, nil, fmt.Errorf("无法写入默认配置: %w", werr)
			}
			// after writing embedded ini, try to read settings
			if loadedCfg, rerr := LoadAppConfig(iniPath); rerr == nil {
				if loadedCfg.OutputDir != "" {
					appCfg.OutputDir = loadedCfg.OutputDir
				}
				appCfg.DownloadProxy = loadedCfg.DownloadProxy
				appCfg.YtDlpURL = loadedCfg.YtDlpURL
			} else {
				return nil, nil, fmt.Errorf("无法读取新创建的配置: %w", rerr)
			}
		} else {
			if werr := SaveAppConfig(iniPath, appCfg); werr != nil {
				return nil, nil, fmt.Errorf("无法保存配置: %w", werr)
			}
		}
	}

	// ensure output directory exists
	if mkerr := os.MkdirAll(appCfg.OutputDir, 0755); mkerr != nil {
		return nil, nil, fmt.Errorf("无法创建输出目录 %s: %w", appCfg.OutputDir, mkerr)
	}

	return appCfg, ytdlpCfg, nil
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
