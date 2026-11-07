package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cstop/internal/collect"
)

var (
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleTabOn  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	styleTabOff = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleHead   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleSel    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

// View 渲染整屏: 状态栏 + tab 栏 + 表体 + 键位栏。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "启动中…"
	}

	var totalRSS int64
	for _, p := range m.procs {
		totalRSS += p.RSS
	}
	head := styleTitle.Render("cstop") +
		fmt.Sprintf(" · 会话 %d · 内存合计 %s", len(m.procs), HumanBytes(totalRSS)) +
		"  " + styleDim.Render(fmt.Sprintf("(%ds 刷新 · %s)", m.cfg.RefreshProcsSec, m.updated.Format("15:04:05")))

	tabs := m.tabsLine()
	body := m.tableBody()
	foot := "q 退出 · 1/2 切换 tab · ↑↓ 滚动 · r 立即刷新"
	if m.errStreak >= 3 {
		foot = styleErr.Render("进程采集连续失败 "+strconv.Itoa(m.errStreak)+" 次,显示旧数据") + " · " + foot
	}
	return head + "\n" + tabs + "\n" + body + "\n" + foot
}

func (m Model) tabsLine() string {
	procsTab := " [1] 运行中 "
	diskTab := " [2] 磁盘 "
	if m.tab == tabProcs {
		procsTab, diskTab = styleTabOn.Render(procsTab), styleTabOff.Render(diskTab)
	} else {
		procsTab, diskTab = styleTabOff.Render(procsTab), styleTabOn.Render(diskTab)
	}
	return procsTab + diskTab
}

// tableBody 渲染当前 tab 的表体(含表头行与滚动窗口)。
func (m Model) tableBody() string {
	// 可用行数 = 总高 - 状态栏 - tab 栏 - 键位栏;磁盘 tab 另留合计行。
	rowsAvail := m.height - 4
	if m.tab == tabDisk {
		rowsAvail--
	}
	var b strings.Builder
	if m.tab == tabProcs {
		b.WriteString(styleHead.Render(fmt.Sprintf("  %-7s %-24s %8s  %9s  %s",
			"PID", "项目", "内存", "运行时长", "最近活动")))
		start, end := visibleWindow(len(m.procs), m.cur, max(rowsAvail-1, 0))
		for i := start; i < end; i++ {
			p := m.procs[i]
			line := fmt.Sprintf("  %-7d %-24s %8s  %9s  %s",
				p.PID, Fit(p.DisplayName(), 24), HumanBytes(p.RSS),
				HumanSince(p.StartedAt), orDash(p.LatestActivity))
			if i == m.cur {
				line = styleSel.Render(line)
			}
			b.WriteString("\n" + line)
		}
		if len(m.procs) == 0 {
			b.WriteString("\n" + styleDim.Render("  无运行中的 Claude Code 会话"))
		}
	} else {
		b.WriteString(styleHead.Render(fmt.Sprintf("  %-46s %4s %10s  %s",
			"项目", "会话", "占用", "最近活动")))
		start, end := visibleWindow(len(m.projects), m.cur, max(rowsAvail-1, 0))
		for i := start; i < end; i++ {
			pr := m.projects[i]
			line := fmt.Sprintf("  %-46s %4d %10s  %s",
				Fit(pr.Name, 46), pr.Sessions, HumanBytes(pr.Bytes),
				pr.LastActive.Format("01-02 15:04"))
			if i == m.cur {
				line = styleSel.Render(line)
			}
			b.WriteString("\n" + line)
		}
		if len(m.projects) == 0 {
			b.WriteString("\n" + styleDim.Render("  无会话记录(或 ~/.claude/projects 不可读)"))
		}
		b.WriteString("\n" + styleDim.Render(fmt.Sprintf("  %-46s %4s %10s",
			"合计", strconv.Itoa(sessionSum(m.projects)), HumanBytes(m.totalDisk))))
	}
	for _, e := range m.diskErrs {
		b.WriteString("\n" + styleErr.Render("  "+e))
	}
	return b.String()
}

func sessionSum(ps []collect.ProjectDisk) int {
	n := 0
	for _, p := range ps {
		n += p.Sessions
	}
	return n
}

// visibleWindow 计算滚动窗口 [start, end): 窗口跟随 cur,保证当前行可见。
func visibleWindow(total, cur, h int) (int, int) {
	if h <= 0 || total == 0 {
		return 0, 0
	}
	if cur >= total {
		cur = total - 1
	}
	if cur < 0 {
		cur = 0
	}
	start := 0
	if cur >= h {
		start = cur - h + 1
	}
	end := start + h
	if end > total {
		end = total
		start = max(0, end-h)
	}
	return start, end
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
