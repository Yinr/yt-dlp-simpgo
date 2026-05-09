package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	nativeDialog "github.com/sqweek/dialog"
)

// showSettingsDialog shows the settings dialog with tabs
func showSettingsDialog(w fyne.Window, appCfg *AppConfig, ytdlpCfg *YTDLPConfig, exeDir string, onSaved func(*AppConfig, *YTDLPConfig)) {
	// Create input bindings for app config
	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetText(appCfg.OutputDir)
	outputDirEntry.Disable()

	downloadProxyEntry := widget.NewEntry()
	downloadProxyEntry.SetText(appCfg.DownloadProxy)
	downloadProxyEntry.SetPlaceHolder("留空则不使用代理，例如：http://127.0.0.1:7890")

	ytDlpURLEntry := widget.NewEntry()
	ytDlpURLEntry.SetText(appCfg.YtDlpURL)
	ytDlpURLEntry.SetPlaceHolder("留空则使用官方地址")

	// Create input bindings for yt-dlp config
	useProxyCheck := widget.NewCheck("启用代理", func(b bool) {
		ytdlpCfg.UseProxy = b
	})
	useProxyCheck.Checked = ytdlpCfg.UseProxy

	proxyAddressEntry := widget.NewEntry()
	proxyAddressEntry.SetText(ytdlpCfg.ProxyAddress)
	proxyAddressEntry.SetPlaceHolder("例如：127.0.0.1:7890 或 http://127.0.0.1:7890")

	syncProxyBtn := widget.NewButton("同步程序代理", func() {
		syncProxyFromApp(appCfg, ytdlpCfg)
		proxyAddressEntry.SetText(ytdlpCfg.ProxyAddress)
		useProxyCheck.Checked = ytdlpCfg.UseProxy
	})
	syncProxyBtn.Importance = widget.MediumImportance

	writeSubsCheck := widget.NewCheck("下载字幕", func(b bool) {
		ytdlpCfg.WriteSubs = b
	})
	writeSubsCheck.Checked = ytdlpCfg.WriteSubs

	subLangsEntry := widget.NewEntry()
	subLangsEntry.SetText(ytdlpCfg.SubLangs)
	subLangsEntry.SetPlaceHolder("例如：chs,cht,zh.* 或 all")

	embedSubsCheck := widget.NewCheck("内嵌字幕", func(b bool) {
		ytdlpCfg.EmbedSubs = b
	})
	embedSubsCheck.Checked = ytdlpCfg.EmbedSubs

	embedChaptersCheck := widget.NewCheck("内嵌章节", func(b bool) {
		ytdlpCfg.EmbedChapters = b
	})
	embedChaptersCheck.Checked = ytdlpCfg.EmbedChapters

	splitChaptersCheck := widget.NewCheck("拆分章节", func(b bool) {
		ytdlpCfg.SplitChapters = b
	})
	splitChaptersCheck.Checked = ytdlpCfg.SplitChapters

	embedMetadataCheck := widget.NewCheck("内嵌元数据", func(b bool) {
		ytdlpCfg.EmbedMetadata = b
	})
	embedMetadataCheck.Checked = ytdlpCfg.EmbedMetadata

	mergeFormatSelect := widget.NewSelect([]string{"mp4/mkv", "mp4", "mkv", "自动（保持原格式）"}, func(selected string) {
		if selected == "自动（保持原格式）" {
			ytdlpCfg.MergeOutputFormat = "auto"
		} else {
			ytdlpCfg.MergeOutputFormat = selected
		}
	})
	if ytdlpCfg.MergeOutputFormat == "auto" {
		mergeFormatSelect.SetSelected("自动（保持原格式）")
	} else {
		mergeFormatSelect.SetSelected(ytdlpCfg.MergeOutputFormat)
	}

	extraArgsEntry := widget.NewMultiLineEntry()
	extraArgsEntry.SetText(ytdlpCfg.ExtraArgs)
	extraArgsEntry.SetPlaceHolder("输入其他 yt-dlp 命令行参数，每行一个，支持注释\n例如：\n--concurrent-fragments 4\n--fragment-retries 10\n\n# 这是注释，不会被解析")

	// Tab 1: App Settings
	chooseDirBtn := widget.NewButton("选择...", func() {
		startDir := appCfg.OutputDir
		if !filepath.IsAbs(startDir) {
			startDir = filepath.Join(exeDir, startDir)
		}
		p, err := nativeDialog.Directory().Title("选择下载目录").SetStartDir(startDir).Browse()
		if err != nil || p == "" {
			return
		}
		if rel, rerr := filepath.Rel(exeDir, p); rerr == nil && !strings.HasPrefix(rel, "..") {
			appCfg.OutputDir = rel
		} else {
			appCfg.OutputDir = p
		}
		outputDirEntry.SetText(appCfg.OutputDir)
	})

	appSettingsContent := container.NewVBox(
		container.NewPadded(widget.NewCard("下载目录", "", container.NewBorder(nil, nil, chooseDirBtn, outputDirEntry, nil))),
		container.NewPadded(widget.NewCard("下载代理（用于下载/更新 yt-dlp）", "", downloadProxyEntry)),
		container.NewPadded(widget.NewCard("yt-dlp 自定义下载 URL", "", ytDlpURLEntry)),
		layout.NewSpacer(),
	)
	appSettingsTab := container.NewScroll(appSettingsContent)

	// Tab 2: yt-dlp Basic Settings
	basicSettingsContent := container.NewVBox(
		container.NewPadded(widget.NewCard("代理设置", "", container.NewVBox(
			useProxyCheck,
			widget.NewLabel("代理地址："),
			container.NewHBox(proxyAddressEntry, syncProxyBtn),
		))),
		container.NewPadded(widget.NewCard("字幕设置", "", container.NewVBox(
			writeSubsCheck,
			widget.NewLabel("字幕语言："),
			subLangsEntry,
			embedSubsCheck,
		))),
		container.NewPadded(widget.NewCard("其他选项", "", container.NewVBox(
			embedChaptersCheck,
			splitChaptersCheck,
			embedMetadataCheck,
			container.NewVBox(widget.NewLabel("输出格式："), mergeFormatSelect),
		))),
		layout.NewSpacer(),
	)
	basicSettingsTab := container.NewScroll(basicSettingsContent)

	// Tab 3: yt-dlp Advanced Settings
	extraArgsEntry.SetMinRowsVisible(8)
	advancedSettingsContent := container.NewVBox(
		container.NewPadded(widget.NewCard("额外参数", "", container.NewVBox(
			widget.NewLabel("输入其他 yt-dlp 命令行参数："),
			widget.NewLabel("每行一个参数，支持注释"),
			extraArgsEntry,
		))),
		layout.NewSpacer(),
	)
	advancedSettingsTab := container.NewScroll(advancedSettingsContent)

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("程序设置", appSettingsTab),
		container.NewTabItem("基础选项", basicSettingsTab),
		container.NewTabItem("高级选项", advancedSettingsTab),
	)

	// Create save button first (will be used in dialog)
	saveBtn := widget.NewButton("保存", nil)
	cancelBtn := widget.NewButton("取消", nil)

	// Create dialog content - use border layout so tabs take available space
	content := container.NewBorder(
		tabs,
		container.NewPadded(
			container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn),
		),
		nil, nil, nil,
	)

	// Create and show dialog
	dlg := dialog.NewCustom("设置", "", content, w)
	dlg.Resize(fyne.NewSize(750, 700))

	// Now set button handlers (after dialog is created)
	cancelBtn.OnTapped = func() {
		dlg.Hide()
	}
	saveBtn.OnTapped = func() {
		// Validate inputs
		if useProxyCheck.Checked && strings.TrimSpace(proxyAddressEntry.Text) == "" {
			dialog.ShowError(fmt.Errorf("启用代理时必须设置代理地址"), w)
			return
		}
		if writeSubsCheck.Checked && strings.TrimSpace(subLangsEntry.Text) == "" {
			dialog.ShowError(fmt.Errorf("下载字幕时必须设置字幕语言"), w)
			return
		}

		// Update configurations
		newAppCfg := &AppConfig{
			OutputDir:     strings.TrimSpace(outputDirEntry.Text),
			DownloadProxy: strings.TrimSpace(downloadProxyEntry.Text),
			YtDlpURL:      strings.TrimSpace(ytDlpURLEntry.Text),
		}

		mergeFormat := mergeFormatSelect.Selected
		if mergeFormat == "自动（保持原格式）" {
			mergeFormat = "auto"
		}

		newYtdlpCfg := &YTDLPConfig{
			UseProxy:          useProxyCheck.Checked,
			ProxyAddress:      strings.TrimSpace(proxyAddressEntry.Text),
			WriteSubs:         writeSubsCheck.Checked,
			SubLangs:          strings.TrimSpace(subLangsEntry.Text),
			EmbedSubs:         embedSubsCheck.Checked,
			EmbedChapters:     embedChaptersCheck.Checked,
			SplitChapters:     splitChaptersCheck.Checked,
			EmbedMetadata:     embedMetadataCheck.Checked,
			MergeOutputFormat: mergeFormat,
			ExtraArgs:         extraArgsEntry.Text,
		}

		// Save configurations
		if err := SaveAppConfig(IniFileName, newAppCfg); err != nil {
			dialog.ShowError(fmt.Errorf("无法保存程序配置: %v", err), w)
			return
		}
		if err := SaveYTDLPConf(YTDLPConfName, newYtdlpCfg); err != nil {
			dialog.ShowError(fmt.Errorf("无法保存yt-dlp配置: %v", err), w)
			return
		}

		// Ensure output directory exists
		actualPath := newAppCfg.OutputDir
		if !filepath.IsAbs(actualPath) {
			actualPath = filepath.Join(exeDir, actualPath)
		}
		if err := os.MkdirAll(actualPath, 0755); err != nil {
			dialog.ShowError(fmt.Errorf("无法创建输出目录: %v", err), w)
			return
		}

		// Call onSaved callback
		if onSaved != nil {
			onSaved(newAppCfg, newYtdlpCfg)
		}

		dlg.Hide()
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "设置已保存",
			Content: "配置已成功保存",
		})
	}

	dlg.Show()
}

// syncProxyFromApp syncs the app download proxy to yt-dlp proxy
func syncProxyFromApp(appCfg *AppConfig, ytdlpCfg *YTDLPConfig) {
	if appCfg.DownloadProxy != "" {
		ytdlpCfg.ProxyAddress = appCfg.DownloadProxy
		ytdlpCfg.UseProxy = true
	} else {
		ytdlpCfg.UseProxy = false
		ytdlpCfg.ProxyAddress = ""
	}
}
