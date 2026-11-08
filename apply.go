package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// RenderPreview 基于当前已保存的 config 渲染将要写入的 shell 配置（不写盘）。
func RenderPreview() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	return cfg.RenderShellConfig(), nil
}

// ApplyResult 描述一次应用配置的结果。
type ApplyResult struct {
	ConfigPath      string `json:"configPath"`
	GeneratedPath   string `json:"generatedPath"`
	PlistPath       string `json:"plistPath"`
	StartInterval   int    `json:"startInterval"`
	LaunchdReloaded bool   `json:"launchdReloaded"`
	ReloadLog       string `json:"reloadLog"`
	Preview         string `json:"preview"`
}

// Apply 把已保存的 config 渲染写入 startAllSs.config，并同步 launchd 自愈周期。
func Apply() (*ApplyResult, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	res := &ApplyResult{
		ConfigPath:    configPath(),
		GeneratedPath: generatedConfigPath(),
		PlistPath:    plistPath(),
		StartInterval: cfg.Global.AutoFixIntervalSec,
		Preview:       cfg.RenderShellConfig(),
	}

	// 1) 写 shell 配置文件（脚本 source 它）
	if err := os.MkdirAll(filepath.Dir(res.GeneratedPath), 0o755); err != nil {
		return res, fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.WriteFile(res.GeneratedPath, []byte(res.Preview), 0o644); err != nil {
		return res, fmt.Errorf("写入 startAllSs.config 失败: %w", err)
	}

	// 2) 同步 launchd StartInterval
	if err := SetStartInterval(cfg.Global.AutoFixIntervalSec); err != nil {
		return res, fmt.Errorf("更新 plist StartInterval 失败: %w", err)
	}

	// 3) 重载 launchd（让新周期立即生效）
	log, err := Reload()
	res.ReloadLog = log
	if err != nil {
		return res, fmt.Errorf("重载 launchd 失败: %w", err)
	}
	res.LaunchdReloaded = true
	return res, nil
}
