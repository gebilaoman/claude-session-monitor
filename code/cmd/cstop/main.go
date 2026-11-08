// cstop — Claude Code 会话监控工具。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
	"cstop/internal/ui"
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

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "home 目录不可得: "+err.Error())
		os.Exit(1)
	}
	root := filepath.Join(home, ".claude", "projects")

	if !*dump {
		for _, w := range warnings {
			fmt.Println("配置: " + w)
		}
		p := tea.NewProgram(ui.New(cfg, root, os.Getpid()), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "TUI 启动失败: "+err.Error())
			os.Exit(1)
		}
		return
	}

	started := time.Now()
	snap := collect.Collect(os.Getpid(), root)
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
		len(snap.Procs), cfg.WarnSessionCount, ui.HumanBytes(totalRSS),
		warn(ui.MB(totalRSS) > cfg.WarnTotalRSSMB), countMark)
	if len(snap.Procs) > 0 {
		fmt.Fprintf(out, "  %-7s %-24s %8s  %9s  %s\n", "PID", "项目", "内存", "运行时长", "最近活动")
		for _, p := range snap.Procs {
			fmt.Fprintf(out, "  %-7d %-24s %8s  %9s  %s\n",
				p.PID, ui.Fit(p.DisplayName(), 24),
				ui.HumanBytes(p.RSS)+warn(ui.MB(p.RSS) > cfg.WarnSessionRSSMB),
				ui.HumanSince(p.StartedAt), orDash(p.LatestActivity))
		}
	}
	fmt.Fprintln(out)

	totalMark := warn(ui.MB(snap.TotalDisk) > cfg.WarnDiskTotalMB)
	fmt.Fprintf(out, "磁盘 %d 项(项目 %d),合计 %s%s\n",
		len(snap.Projects), projectCount(snap.Projects), ui.HumanBytes(snap.TotalDisk), totalMark)
	if len(snap.Projects) > 0 {
		fmt.Fprintf(out, "  %-46s %4s %10s  %s\n", "项目", "会话", "占用", "最近活动")
		for _, pr := range snap.Projects {
			sess := fmt.Sprintf("%d", pr.Sessions)
			if pr.Kind != "project" {
				sess = "-"
			}
			fmt.Fprintf(out, "  %-46s %4s %10s  %s\n",
				ui.Fit(pr.Name, 46), sess,
				ui.HumanBytes(pr.Bytes)+warn(ui.MB(pr.Bytes) > cfg.WarnDiskProjectMB),
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

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func projectCount(ps []collect.ProjectDisk) int {
	n := 0
	for _, p := range ps {
		if p.Kind == "project" {
			n++
		}
	}
	return n
}

func warn(over bool) string {
	if over {
		return " ⚠"
	}
	return ""
}
