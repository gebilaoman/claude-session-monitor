// cstop — Claude Code 会话监控工具。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cstop/internal/collect"
	"cstop/internal/config"
)

// version 由构建注入(CI ldflags -X main.version=<tag>);本地构建为 dev。
var version = "dev"

func main() {
	refresh := flag.Int("refresh", 0, "进程刷新间隔(秒),覆盖配置文件")
	configPath := flag.String("config", "", "配置文件路径;默认 os.UserConfigDir()/cstop/config.toml")
	initConfig := flag.Bool("init-config", false, "生成默认配置文件后退出")
	dump := flag.Bool("dump", false, "打印一份快照文本,不启动 TUI")
	showVersion := flag.Bool("version", false, "打印版本号后退出")
	flag.Parse()

	if *showVersion {
		fmt.Println("cstop " + version)
		return
	}

	path := *configPath
	if path == "" {
		p, err := config.Path()
		if err != nil {
			fmt.Fprintln(os.Stderr, "配置路径不可得: "+err.Error())
			os.Exit(1)
		}
		path = p
	}
	if *initConfig {
		if err := config.WriteDefault(path); err != nil {
			fmt.Fprintln(os.Stderr, "生成配置失败: "+err.Error())
			os.Exit(1)
		}
		fmt.Println("已生成: " + path)
		return
	}

	cfg, warnings := config.Load(path)
	if *refresh > 0 {
		cfg.RefreshProcsSec = *refresh
	}

	if !*dump {
		fmt.Println("TUI 将在 M2 提供。当前版本请使用: cstop --dump")
		for _, w := range warnings {
			fmt.Println("配置: " + w)
		}
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "home 目录不可得: "+err.Error())
		os.Exit(1)
	}
	started := time.Now()
	snap := collect.Collect(os.Getpid(), filepath.Join(home, ".claude", "projects"))
	printDump(cfg, snap, warnings, started)
}

func printDump(cfg config.Config, snap collect.Snapshot, configWarnings []string, started time.Time) {
	out := os.Stdout
	fmt.Fprintf(out, "cstop %s · 采集耗时 %dms\n\n", version, time.Since(started).Milliseconds())

	var totalRSS int64
	for _, p := range snap.Procs {
		totalRSS += p.RSS
	}
	countMark := warn(len(snap.Procs) > cfg.WarnSessionCount)
	fmt.Fprintf(out, "运行中会话 %d 个(告警线 %d),内存合计 %s%s%s\n",
		len(snap.Procs), cfg.WarnSessionCount, humanBytes(totalRSS),
		warn(mb(totalRSS) > cfg.WarnTotalRSSMB), countMark)
	if len(snap.Procs) > 0 {
		fmt.Fprintf(out, "  %-7s %-24s %8s  %9s  %s\n", "PID", "项目", "内存", "运行时长", "最近活动")
		for _, p := range snap.Procs {
			fmt.Fprintf(out, "  %-7d %-24s %8s  %9s  %s\n",
				p.PID, fit(displayProject(p), 24), humanBytes(p.RSS)+warn(mb(p.RSS) > cfg.WarnSessionRSSMB),
				humanSince(p.StartedAt), orDash(p.LatestActivity))
		}
	}
	fmt.Fprintln(out)

	totalMark := warn(mb(snap.TotalDisk) > cfg.WarnDiskTotalMB)
	fmt.Fprintf(out, "磁盘 %d 个项目,合计 %s%s\n", len(snap.Projects), humanBytes(snap.TotalDisk), totalMark)
	if len(snap.Projects) > 0 {
		fmt.Fprintf(out, "  %-46s %4s %10s  %s\n", "项目", "会话", "占用", "最近活动")
		for _, pr := range snap.Projects {
			fmt.Fprintf(out, "  %-46s %4d %10s  %s\n",
				fit(pr.Name, 46), pr.Sessions,
				humanBytes(pr.Bytes)+warn(mb(pr.Bytes) > cfg.WarnDiskProjectMB),
				pr.LastActive.Format("01-02 15:04"))
		}
	}
	fmt.Fprintln(out)

	for _, w := range configWarnings {
		fmt.Fprintln(out, "配置: "+w)
	}
	for _, e := range snap.Errors {
		fmt.Fprintln(out, "采集: "+e)
	}
}

// displayProject 取进程的项目显示名: cwd 末段;取不到回退 --resume 的 id 或 -。
func displayProject(p collect.Proc) string {
	if p.ProjectDir != "" {
		return filepath.Base(p.ProjectDir)
	}
	if p.SessionID != "" {
		return p.SessionID
	}
	return "-"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func warn(over bool) string {
	if over {
		return " ⚠"
	}
	return ""
}

// fit 截断过长显示名,保持列对齐。
func fit(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func mb(b int64) int { return int(b / 1024 / 1024) }

func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%dMB", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%dKB", b/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func humanSince(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}
