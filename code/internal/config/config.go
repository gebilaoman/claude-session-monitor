// Package config 加载与生成 cstop 配置。
// 配置为平面 TOML 子集: 每行 key = int 或 bool,# 起注释(整行或行尾)。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 顶层刷新间隔与阈值。单位见字段名。
type Config struct {
	RefreshProcsSec    int
	RefreshDiskSec     int
	WarnSessionRSSMB   int
	WarnTotalRSSMB     int
	WarnDiskProjectMB  int
	WarnDiskTotalMB    int
	WarnSessionCount   int
	UpdateCheckOnStart bool
}

// Default 返回内置默认值。取自 2026-11-08 本机实测。
func Default() Config {
	return Config{
		RefreshProcsSec:    2,
		RefreshDiskSec:     60,
		WarnSessionRSSMB:   400,
		WarnTotalRSSMB:     1500,
		WarnDiskProjectMB:  500,
		WarnDiskTotalMB:    2000,
		WarnSessionCount:   6,
		UpdateCheckOnStart: true,
	}
}

// Path 返回默认配置文件路径: os.UserConfigDir()/cstop/config.toml。
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cstop", "config.toml"), nil
}

// Load 读取配置。文件不存在或个别行损坏时用默认值,warnings 说明原因。
func Load(path string) (cfg Config, warnings []string) {
	cfg = Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, []string{"配置读取失败,用默认值: " + err.Error()}
	}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			warnings = append(warnings, fmt.Sprintf("配置第 %d 行无 '=',已忽略", i+1))
			continue
		}
		key = strings.TrimSpace(key)
		val, _, _ = strings.Cut(strings.TrimSpace(val), "#")
		applyLine(&cfg, key, strings.TrimSpace(val), i+1, &warnings)
	}
	return cfg, warnings
}

func applyLine(cfg *Config, key, val string, lineNo int, warnings *[]string) {
	intField := func(dst *int) {
		n, err := strconv.Atoi(val)
		if err != nil || n < 0 {
			*warnings = append(*warnings, fmt.Sprintf("配置第 %d 行 %s 值非法,用默认值", lineNo, key))
			return
		}
		*dst = n
	}
	switch key {
	case "refresh_procs_sec":
		intField(&cfg.RefreshProcsSec)
	case "refresh_disk_sec":
		intField(&cfg.RefreshDiskSec)
	case "warn_session_rss_mb":
		intField(&cfg.WarnSessionRSSMB)
	case "warn_total_rss_mb":
		intField(&cfg.WarnTotalRSSMB)
	case "warn_disk_project_mb":
		intField(&cfg.WarnDiskProjectMB)
	case "warn_disk_total_mb":
		intField(&cfg.WarnDiskTotalMB)
	case "warn_session_count":
		intField(&cfg.WarnSessionCount)
	case "update_check_on_start":
		b, err := strconv.ParseBool(val)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("配置第 %d 行 %s 值非法,用默认值", lineNo, key))
			return
		}
		cfg.UpdateCheckOnStart = b
	default:
		*warnings = append(*warnings, fmt.Sprintf("配置第 %d 行未知键 %s,已忽略", lineNo, key))
	}
}

// WriteDefault 生成默认配置文件(含父目录)。文件已存在时报错。
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("已存在: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	d := Default()
	body := fmt.Sprintf(`# cstop 配置。删除本文件即恢复全部默认值。
# 生成: cstop --init-config

refresh_procs_sec = %d     # 进程与活动刷新间隔(秒)
refresh_disk_sec = %d      # 磁盘扫描间隔(秒)
warn_session_rss_mb = %d   # 单会话内存告警线
warn_total_rss_mb = %d     # 全部会话内存合计告警线
warn_disk_project_mb = %d  # 单项目磁盘告警线
warn_disk_total_mb = %d    # projects 总量告警线
warn_session_count = %d    # 运行会话数告警线
update_check_on_start = %t # 启动时后台检测新版本
`, d.RefreshProcsSec, d.RefreshDiskSec, d.WarnSessionRSSMB, d.WarnTotalRSSMB,
		d.WarnDiskProjectMB, d.WarnDiskTotalMB, d.WarnSessionCount, d.UpdateCheckOnStart)
	return os.WriteFile(path, []byte(body), 0o644)
}
