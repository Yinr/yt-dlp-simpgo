package main

import (
	"os"

	_ "embed"

	ini "gopkg.in/ini.v1"
)

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

// LoadConfig loads only the program settings from the ini file.
// It returns the configured output directory (or empty string if not set).
func LoadConfig(path string) (outputDir string, err error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return "", nil
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return "", err
	}
	outputDir = cfg.Section("app").Key("output_dir").MustString("")
	return outputDir, nil
}

// SaveConfig writes only the program settings (currently output_dir) into the ini file.
func SaveConfig(path, outputDir string) error {
	cfg := ini.Empty()
	app := cfg.Section("app")
	app.Key("output_dir").SetValue(outputDir)
	return cfg.SaveTo(path)
}

// EnsureDefaults ensures yt-dlp.conf and the ini file exist. It will:
//   - create yt-dlp.conf from embedded defaultYTDLPConf if the file is missing;
//   - create the ini file from embedded defaultIniConf if missing, or write a minimal
//     ini containing output_dir when no embedded ini is available.
//
// It returns the resolved outputDir (either the provided defaultOutputDir or the
// value read from an existing ini) and any error encountered.
func EnsureDefaults(iniPath, defaultOutputDir string) (outputDir string, err error) {
	outputDir = defaultOutputDir

	// If yt-dlp.conf doesn't exist, write embedded default
	if _, st := os.Stat(YTDLPConfName); os.IsNotExist(st) {
		if defaultYTDLPConf != "" {
			if werr := os.WriteFile(YTDLPConfName, []byte(defaultYTDLPConf), 0644); werr != nil {
				return outputDir, werr
			}
		}
	}

	// If ini exists, read output_dir from it
	if _, serr := os.Stat(iniPath); serr == nil {
		if od, rerr := LoadConfig(iniPath); rerr == nil {
			if od != "" {
				outputDir = od
			}
		} else {
			// return read error
			return outputDir, rerr
		}
	} else {
		// ini missing: if we have an embedded ini template, write it; otherwise write minimal ini
		if defaultIniConf != "" {
			if werr := os.WriteFile(iniPath, []byte(defaultIniConf), 0644); werr != nil {
				return outputDir, werr
			}
			// after writing embedded ini, try to read output_dir
			if od, rerr := LoadConfig(iniPath); rerr == nil && od != "" {
				outputDir = od
			}
		} else {
			if werr := SaveConfig(iniPath, outputDir); werr != nil {
				return outputDir, werr
			}
		}
	}

	// ensure output directory exists
	if mkerr := os.MkdirAll(outputDir, 0755); mkerr != nil {
		return outputDir, mkerr
	}

	return outputDir, nil
}
