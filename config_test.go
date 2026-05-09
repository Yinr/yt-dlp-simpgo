package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultYTDLPConfig(t *testing.T) {
	cfg := DefaultYTDLPConfig()

	if !cfg.WriteSubs {
		t.Error("默认应启用下载字幕")
	}
	if cfg.SubLangs != "chs,cht,zh.*" {
		t.Errorf("默认字幕语言应为 chs,cht,zh.*，实际为 %s", cfg.SubLangs)
	}
	if !cfg.EmbedSubs {
		t.Error("默认应启用内嵌字幕")
	}
	if !cfg.EmbedChapters {
		t.Error("默认应启用内嵌章节")
	}
	if !cfg.EmbedMetadata {
		t.Error("默认应启用内嵌元数据")
	}
	if cfg.MergeOutputFormat != "mp4/mkv" {
		t.Errorf("默认合并格式应为 mp4/mkv，实际为 %s", cfg.MergeOutputFormat)
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ini")

	cfg := &AppConfig{
		OutputDir:     "./下载",
		DownloadProxy: "http://127.0.0.1:7890",
		YtDlpURL:      "https://example.com/yt-dlp.exe",
	}

	if err := SaveAppConfig(path, cfg); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	loaded, err := LoadAppConfig(path)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if loaded.OutputDir != cfg.OutputDir {
		t.Errorf("OutputDir 不匹配: %s != %s", loaded.OutputDir, cfg.OutputDir)
	}
	if loaded.DownloadProxy != cfg.DownloadProxy {
		t.Errorf("DownloadProxy 不匹配: %s != %s", loaded.DownloadProxy, cfg.DownloadProxy)
	}
	if loaded.YtDlpURL != cfg.YtDlpURL {
		t.Errorf("YtDlpURL 不匹配: %s != %s", loaded.YtDlpURL, cfg.YtDlpURL)
	}
}

func TestLoadAppConfig_NotExists(t *testing.T) {
	cfg, err := LoadAppConfig("nonexistent.ini")
	if err != nil {
		t.Fatalf("加载不存在的配置不应报错: %v", err)
	}
	if cfg == nil {
		t.Fatal("返回的配置不应为 nil")
	}
}

func TestSaveAndLoadYTDLPConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "yt-dlp.conf")

	cfg := &YTDLPConfig{
		UseProxy:          true,
		ProxyAddress:      "127.0.0.1:7890",
		WriteSubs:         true,
		SubLangs:          "all",
		EmbedSubs:         true,
		EmbedChapters:     false,
		SplitChapters:     true,
		EmbedMetadata:     true,
		MergeOutputFormat: "mkv",
		ExtraArgs:         "--concurrent-fragments 4",
	}

	if err := SaveYTDLPConf(path, cfg); err != nil {
		t.Fatalf("保存 yt-dlp 配置失败: %v", err)
	}

	loaded, err := LoadYTDLPConf(path)
	if err != nil {
		t.Fatalf("加载 yt-dlp 配置失败: %v", err)
	}

	if loaded.UseProxy != cfg.UseProxy {
		t.Errorf("UseProxy 不匹配: %v != %v", loaded.UseProxy, cfg.UseProxy)
	}
	if loaded.ProxyAddress != cfg.ProxyAddress {
		t.Errorf("ProxyAddress 不匹配: %s != %s", loaded.ProxyAddress, cfg.ProxyAddress)
	}
	if loaded.WriteSubs != cfg.WriteSubs {
		t.Errorf("WriteSubs 不匹配: %v != %v", loaded.WriteSubs, cfg.WriteSubs)
	}
	if loaded.SubLangs != cfg.SubLangs {
		t.Errorf("SubLangs 不匹配: %s != %s", loaded.SubLangs, cfg.SubLangs)
	}
	if loaded.EmbedSubs != cfg.EmbedSubs {
		t.Errorf("EmbedSubs 不匹配: %v != %v", loaded.EmbedSubs, cfg.EmbedSubs)
	}
	if loaded.EmbedChapters != cfg.EmbedChapters {
		t.Errorf("EmbedChapters 不匹配: %v != %v", loaded.EmbedChapters, cfg.EmbedChapters)
	}
	if loaded.SplitChapters != cfg.SplitChapters {
		t.Errorf("SplitChapters 不匹配: %v != %v", loaded.SplitChapters, cfg.SplitChapters)
	}
	if loaded.EmbedMetadata != cfg.EmbedMetadata {
		t.Errorf("EmbedMetadata 不匹配: %v != %v", loaded.EmbedMetadata, cfg.EmbedMetadata)
	}
	if loaded.MergeOutputFormat != cfg.MergeOutputFormat {
		t.Errorf("MergeOutputFormat 不匹配: %s != %s", loaded.MergeOutputFormat, cfg.MergeOutputFormat)
	}
}

func TestLoadYTDLPConf_NotExists(t *testing.T) {
	cfg, err := LoadYTDLPConf("nonexistent.conf")
	if err != nil {
		t.Fatalf("加载不存在的配置不应报错: %v", err)
	}
	if cfg == nil {
		t.Fatal("返回的配置不应为 nil")
	}
	if !cfg.WriteSubs {
		t.Error("默认配置应启用下载字幕")
	}
}

func TestEnsureDefaults(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "test.ini")

	// 切换到临时目录以测试文件创建
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	appCfg, ytdlpCfg, err := EnsureDefaults(iniPath, filepath.Join(dir, "下载"))
	if err != nil {
		t.Fatalf("EnsureDefaults 失败: %v", err)
	}

	if appCfg == nil {
		t.Fatal("appCfg 不应为 nil")
	}
	if ytdlpCfg == nil {
		t.Fatal("ytdlpCfg 不应为 nil")
	}

	// 验证输出目录已创建
	if _, err := os.Stat(appCfg.OutputDir); os.IsNotExist(err) {
		t.Error("输出目录应被创建")
	}
}
