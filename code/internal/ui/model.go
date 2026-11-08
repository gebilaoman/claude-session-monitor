// Package ui 用 Bubble Tea 渲染 cstop 的终端界面。
package ui

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
	"cstop/internal/update"
)

const (
	tabProcs = iota
	tabDisk
)

// Model cstop TUI 状态。
type Model struct {
	cfg     config.Config
	root    string
	selfPID int
	version string

	tab      int // tabProcs / tabDisk
	cur      int // 当前行(或详情页滚动行)
	width    int
	height   int
	quitting bool

	procs     []collect.Proc
	projects  []collect.ProjectDisk
	totalDisk int64

	detail      bool         // 详情页打开
	detailProc  collect.Proc // 详情对应的进程
	detailLines []string     // 最近活动(最新在前)

	namePrefix string // 项目 slug 的 home 前缀(展示时剥离)

	procErrs  []string
	diskErrs  []string
	errStreak int // 进程采集连续失败次数;>=3 时状态栏报错

	updateAvailable string // 新版本提示,如 "v0.2.0";空串为无
	wantUpdate      bool   // 用户按 u 请求更新(退出后由 main 执行)

	updated time.Time
}

// New 装配 TUI。刷新间隔下限 1 秒;version 为构建注入版本号(dev 时跳过更新检测)。
func New(cfg config.Config, root string, selfPID int, version string) Model {
	if cfg.RefreshProcsSec < 1 {
		cfg.RefreshProcsSec = 1
	}
	if cfg.RefreshDiskSec < 1 {
		cfg.RefreshDiskSec = 1
	}
	// 项目 slug 都以 home 路径的 slug 开头;展示时剥离该公共前缀。
	prefix := ""
	if home := filepath.Dir(filepath.Dir(root)); home != "/" && home != "." {
		prefix = collect.SlugFromPath(home) + "-"
	}
	return Model{cfg: cfg, root: root, selfPID: selfPID, version: version, namePrefix: prefix}
}

type procsMsg struct {
	procs []collect.Proc
	errs  []string
	at    time.Time
}

type diskMsg struct {
	projects []collect.ProjectDisk
	total    int64
	errs     []string
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{procsTickNow(m.selfPID, m.root), diskTickNow(m.root)}
	if m.checkUpdates() {
		cmds = append(cmds, updateCheckCmd(m.version), recheckUpdateCmd())
	}
	return tea.Batch(cmds...)
}

// checkUpdates 是否启用新版本检测: 非本地构建且配置允许。
func (m Model) checkUpdates() bool {
	return m.version != "" && m.version != "dev" && m.cfg.UpdateCheckOnStart
}

// updateCheckMsg 新版本检测结果;latest 空串为无新版(或检测失败,静默)。
type updateCheckMsg struct{ latest string }

// updateTickMsg 6 小时重检到点。
type updateTickMsg struct{}

// updateCheckCmd 查询新版本(3 秒超时,失败静默)。
func updateCheckCmd(version string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		available, latest, err := update.Check(ctx, version)
		if err != nil || !available {
			return updateCheckMsg{}
		}
		return updateCheckMsg{latest: latest}
	}
}

// recheckUpdateCmd 每 6 小时重检一次(监控工具长驻,中途发版也能知道)。
func recheckUpdateCmd() tea.Cmd {
	return tea.Tick(6*time.Hour, func(time.Time) tea.Msg { return updateTickMsg{} })
}

func procsTickNow(selfPID int, root string) tea.Cmd {
	return func() tea.Msg {
		procs, errs := collect.ProcsTick(selfPID, root)
		return procsMsg{procs: procs, errs: errs, at: time.Now()}
	}
}

func diskTickNow(root string) tea.Cmd {
	return func() tea.Msg {
		projects, total, errs := collect.DiskTick(root)
		return diskMsg{projects: projects, total: total, errs: errs}
	}
}

func procsTickAfter(d time.Duration, selfPID int, root string) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		procs, errs := collect.ProcsTick(selfPID, root)
		return procsMsg{procs: procs, errs: errs, at: time.Now()}
	})
}

func diskTickAfter(d time.Duration, root string) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		projects, total, errs := collect.DiskTick(root)
		return diskMsg{projects: projects, total: total, errs: errs}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case procsMsg:
		if msg.procs == nil && len(msg.errs) > 0 {
			m.errStreak++ // 枚举失败,保留上一份数据
		} else {
			m.errStreak = 0
			m.procs = msg.procs
			m.procErrs = msg.errs
			m.updated = msg.at
		}
		m.refreshDetail()
		if m.tab == tabProcs && !m.detail && m.cur >= len(m.procs) {
			m.cur = max(0, len(m.procs)-1)
		}
		return m, procsTickAfter(time.Duration(m.cfg.RefreshProcsSec)*time.Second, m.selfPID, m.root)

	case diskMsg:
		m.projects = msg.projects
		m.totalDisk = msg.total
		m.diskErrs = msg.errs
		if m.tab == tabDisk && m.cur >= len(m.projects) {
			m.cur = max(0, len(m.projects)-1)
		}
		return m, diskTickAfter(time.Duration(m.cfg.RefreshDiskSec)*time.Second, m.root)

	case updateCheckMsg:
		if msg.latest != "" {
			m.updateAvailable = msg.latest
		}

	case updateTickMsg:
		if m.checkUpdates() {
			return m, tea.Batch(updateCheckCmd(m.version), recheckUpdateCmd())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.detail {
				m.detail, m.cur = false, 0
			}
		case "u":
			if m.updateAvailable != "" {
				m.wantUpdate = true
				m.quitting = true
				return m, tea.Quit
			}
		case "enter", "ctrl+j": // ICRNL 环境下 Enter 到达为 \n(ctrl+j)
			if !m.detail && m.tab == tabProcs && len(m.procs) > 0 {
				m.openDetail(m.procs[m.cur])
			}
		case "1":
			if !m.detail {
				m.tab, m.cur = tabProcs, 0
			}
		case "2":
			if !m.detail {
				m.tab, m.cur = tabDisk, 0
			}
		case "left":
			if !m.detail && m.tab > tabProcs {
				m.tab, m.cur = m.tab-1, 0
			}
		case "right":
			if !m.detail && m.tab < tabDisk {
				m.tab, m.cur = m.tab+1, 0
			}
		case "up":
			if m.cur > 0 {
				m.cur--
			}
		case "down":
			if m.cur < m.rowCount()-1 {
				m.cur++
			}
		case "r":
			return m, tea.Batch(procsTickNow(m.selfPID, m.root), diskTickNow(m.root))
		}
	}
	return m, nil
}

func (m *Model) openDetail(p collect.Proc) {
	m.detail = true
	m.detailProc = p
	m.detailLines = collect.ActivityHistory(m.detailProjectDir(), 20)
	m.cur = 0
}

// refreshDetail 进程刷新后同步详情页: 进程消失则关闭,仍在则更新数据。
func (m *Model) refreshDetail() {
	if !m.detail {
		return
	}
	for _, p := range m.procs {
		if p.PID == m.detailProc.PID {
			m.detailProc = p
			m.detailLines = collect.ActivityHistory(m.detailProjectDir(), 20)
			if m.cur >= len(m.detailLines) {
				m.cur = max(0, len(m.detailLines)-1)
			}
			return
		}
	}
	m.detail, m.cur = false, 0
}

// detailProjectDir 详情进程对应的 projects 子目录;无 cwd 时为 ""。
func (m *Model) detailProjectDir() string {
	if m.detailProc.ProjectDir == "" {
		return ""
	}
	return filepath.Join(m.root, collect.SlugFromPath(m.detailProc.ProjectDir))
}

func (m Model) rowCount() int {
	if m.detail {
		return len(m.detailLines)
	}
	if m.tab == tabDisk {
		return len(m.projects)
	}
	return len(m.procs)
}

// WantUpdate 用户是否在 TUI 内按 u 请求更新(由 main 在退出后执行)。
func (m Model) WantUpdate() bool { return m.wantUpdate }
