// cstop — Claude Code 会话监控工具。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
	"cstop/internal/ui"
	"cstop/internal/update"
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

	// 子命令: cstop update
	if len(flag.Args()) > 0 && flag.Args()[0] == "update" {
		if version == "dev" {
			fmt.Fprintln(os.Stderr, "本地构建无版本号,无法检测更新;请用 brew 或发布包安装的版本")
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := update.Run(ctx, version); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
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

		// 启动时后台检测新版本(3 秒超时,失败静默),结果注入 TUI。
		if cfg.UpdateCheckOnStart && version != "dev" {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				available, latest, err := update.Check(ctx, version)
				if err == nil && available {
					p.Send(ui.UpdateMsg{Version: latest})
				}
			}()
		}

		final, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "TUI 启动失败: "+err.Error())
			os.Exit(1)
		}
		// 用户在 TUI 内按 u: 退出后在此执行更新流程,进度直接可见。
		if m, ok := final.(ui.Model); ok && m.WantUpdate() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := update.Run(ctx, version); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
		return
	}

	started := time.Now()
	snap := collect.Collect(os.Getpid(), root)
	printDump(cfg, snap, root, warnings, started)
}

func printDump(cfg config.Config, snap collect.Snapshot, root string, configWarnings []string, started time.Time) {
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
		namePrefix := ""
		if home := filepath.Dir(filepath.Dir(root)); home != "/" && home != "." {
			namePrefix = collect.SlugFromPath(home) + "-"
		}
		fmt.Fprintf(out, "  %-46s %4s %10s  %s\n", "项目", "会话", "占用", "最近活动")
		for _, pr := range snap.Projects {
			sess := fmt.Sprintf("%d", pr.Sessions)
			if pr.Kind != "project" {
				sess = "-"
			}
			fmt.Fprintf(out, "  %-46s %4s %10s  %s\n",
				ui.FitTail(strings.TrimPrefix(pr.Name, namePrefix), 46), sess,
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
