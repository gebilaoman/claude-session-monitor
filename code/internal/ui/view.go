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
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // 超限
	styleOver   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // 超限 1.5 倍
)

// View 渲染整屏: 状态栏 + tab 栏 + 表体 + 键位栏;详情页打开时渲染详情。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "启动中…"
	}
	if m.detail {
		return m.detailView()
	}
	return m.mainView()
}

func (m Model) mainView() string {
	head := m.statusLine()
	tabs := m.tabsLine()
	body := m.tableBody()
	foot := "q 退出 · 1/2 切换 tab · ↑↓ 滚动 · r 立即刷新 · Enter 详情"
	if m.errStreak >= 3 {
		foot = styleErr.Render("进程采集连续失败 "+strconv.Itoa(m.errStreak)+" 次,显示旧数据") + " · " + foot
	}
	return head + "\n" + tabs + "\n" + body + "\n" + foot
}

// statusLine 顶部状态栏;任一阈值超限时追加 ⚠。
func (m Model) statusLine() string {
	var totalRSS int64
	for _, p := range m.procs {
		totalRSS += p.RSS
	}
	sess := markCell(fmt.Sprintf("会话 %d", len(m.procs)),
		len(m.procs), m.cfg.WarnSessionCount)
	mem := markCell(fmt.Sprintf("内存合计 %s", HumanBytes(totalRSS)),
		MB(totalRSS), m.cfg.WarnTotalRSSMB)
	anyOver := thresholdLevel(len(m.procs), m.cfg.WarnSessionCount) > 0 ||
		thresholdLevel(MB(totalRSS), m.cfg.WarnTotalRSSMB) > 0 ||
		thresholdLevel(MB(m.totalDisk), m.cfg.WarnDiskTotalMB) > 0
	for _, p := range m.procs {
		if thresholdLevel(MB(p.RSS), m.cfg.WarnSessionRSSMB) > 0 {
			anyOver = true
			break
		}
	}
	for _, pr := range m.projects {
		if thresholdLevel(MB(pr.Bytes), m.cfg.WarnDiskProjectMB) > 0 {
			anyOver = true
			break
		}
	}
	head := styleTitle.Render("cstop") + " · " + sess + " · " + mem
	if anyOver {
		head += " " + styleOver.Render("⚠")
	}
	return head + "  " + styleDim.Render(fmt.Sprintf("(%ds 刷新 · %s)",
		m.cfg.RefreshProcsSec, m.updated.Format("15:04:05")))
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
			mem := markCell(fmt.Sprintf("%8s", HumanBytes(p.RSS)), MB(p.RSS), m.cfg.WarnSessionRSSMB)
			line := fmt.Sprintf("  %-7d %-24s %s  %9s  %s",
				p.PID, Fit(p.DisplayName(), 24), mem,
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
			sz := markCell(fmt.Sprintf("%10s", HumanBytes(pr.Bytes)), MB(pr.Bytes), m.cfg.WarnDiskProjectMB)
			line := fmt.Sprintf("  %-46s %4s %s  %s",
				FitTail(strings.TrimPrefix(pr.Name, m.namePrefix), 46), sessionsCell(pr), sz,
				pr.LastActive.Format("01-02 15:04"))
			if i == m.cur {
				line = styleSel.Render(line)
			}
			b.WriteString("\n" + line)
		}
		if len(m.projects) == 0 {
			b.WriteString("\n" + styleDim.Render("  无会话记录(或 ~/.claude/projects 不可读)"))
		}
		total := markCell(fmt.Sprintf("%10s", HumanBytes(m.totalDisk)), MB(m.totalDisk), m.cfg.WarnDiskTotalMB)
		b.WriteString("\n" + styleDim.Render(fmt.Sprintf("  %-46s %4s %s",
			"合计", strconv.Itoa(sessionSum(m.projects)), total)))
	}
	for _, e := range m.diskErrs {
		b.WriteString("\n" + styleErr.Render("  "+e))
	}
	return b.String()
}

// detailView 单会话详情页: 头部信息 + 最近 20 条活动,Esc 返回。
func (m Model) detailView() string {
	p := m.detailProc
	var b strings.Builder
	b.WriteString(styleTitle.Render(fmt.Sprintf("会话 %d", p.PID)) +
		"  " + styleDim.Render("Esc 返回"))
	b.WriteString("\n  " + kv("项目", orDash(p.ProjectDir)))
	b.WriteString("\n  " + kv("SessionID", orDash(p.SessionID)))
	mem := markCell(HumanBytes(p.RSS), MB(p.RSS), m.cfg.WarnSessionRSSMB)
	b.WriteString("\n  " + kv("内存", mem) + "  " + kv("运行时长", HumanSince(p.StartedAt)))
	b.WriteString("\n  " + styleDim.Render("── 最近活动(最多 20 条,新在前)──"))

	rowsAvail := max(m.height-7, 1)
	start, end := visibleWindow(len(m.detailLines), m.cur, rowsAvail)
	if len(m.detailLines) == 0 {
		b.WriteString("\n  " + styleDim.Render("(无活动记录)"))
	}
	for i := start; i < end; i++ {
		b.WriteString("\n  " + m.detailLines[i])
	}
	return b.String()
}

func kv(k, v string) string {
	return styleDim.Render(fmt.Sprintf("%-10s", k)) + v
}

// thresholdLevel 阈值等级: 0 正常,1 超限(标黄),2 超限 1.5 倍(标红)。
func thresholdLevel(value, warnLine int) int {
	if warnLine <= 0 || value <= warnLine {
		return 0
	}
	if value*2 > warnLine*3 {
		return 2
	}
	return 1
}

// markCell 按阈值等级给已格式化的单元格着色。
func markCell(cell string, value, warnLine int) string {
	switch thresholdLevel(value, warnLine) {
	case 1:
		return styleWarn.Render(cell)
	case 2:
		return styleOver.Render(cell)
	}
	return cell
}

func sessionSum(ps []collect.ProjectDisk) int {
	n := 0
	for _, p := range ps {
		if p.Kind == "project" {
			n += p.Sessions
		}
	}
	return n
}

// sessionsCell 会话数列: 项目显示数量,其他目录/文件显示 -。
func sessionsCell(pr collect.ProjectDisk) string {
	if pr.Kind != "project" {
		return "-"
	}
	return strconv.Itoa(pr.Sessions)
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
