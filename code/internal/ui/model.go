// Package ui 用 Bubble Tea 渲染 cstop 的终端界面。
package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
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

	tab      int // tabProcs / tabDisk
	cur      int // 当前行
	width    int
	height   int
	quitting bool

	procs     []collect.Proc
	projects  []collect.ProjectDisk
	totalDisk int64

	procErrs  []string
	diskErrs  []string
	errStreak int // 进程采集连续失败次数;>=3 时状态栏报错

	updated time.Time
}

// New 装配 TUI。刷新间隔下限 1 秒。
func New(cfg config.Config, root string, selfPID int) Model {
	if cfg.RefreshProcsSec < 1 {
		cfg.RefreshProcsSec = 1
	}
	if cfg.RefreshDiskSec < 1 {
		cfg.RefreshDiskSec = 1
	}
	return Model{cfg: cfg, root: root, selfPID: selfPID}
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
	return tea.Batch(procsTickNow(m.selfPID, m.root), diskTickNow(m.root))
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
		if m.tab == tabProcs && m.cur >= len(m.procs) {
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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.tab, m.cur = tabProcs, 0
		case "2":
			m.tab, m.cur = tabDisk, 0
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

func (m Model) rowCount() int {
	if m.tab == tabDisk {
		return len(m.projects)
	}
	return len(m.procs)
}
