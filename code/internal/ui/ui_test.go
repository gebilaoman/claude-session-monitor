package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
)

func TestVisibleWindow(t *testing.T) {
	cases := []struct {
		total, cur, h int
		start, end    int
	}{
		{100, 0, 10, 0, 10},
		{100, 9, 10, 0, 10},
		{100, 10, 10, 1, 11},
		{100, 99, 10, 90, 100},
		{5, 0, 10, 0, 5},
		{5, 4, 10, 0, 5},
		{0, 0, 10, 0, 0},
		{10, 0, 0, 0, 0},
	}
	for _, c := range cases {
		start, end := visibleWindow(c.total, c.cur, c.h)
		if start != c.start || end != c.end {
			t.Errorf("visibleWindow(%d,%d,%d) = (%d,%d), want (%d,%d)",
				c.total, c.cur, c.h, start, end, c.start, c.end)
		}
	}
}

func testModel() Model {
	m := New(config.Default(), "/tmp/projects", 1)
	m.width, m.height = 120, 30
	m.updated = time.Now()
	m.procs = []collect.Proc{
		{PID: 1423, RSS: 200 << 20, ProjectDir: "/Users/x/work/zelig", StartedAt: time.Now().Add(-time.Hour), LatestActivity: "Bash · 12:05:00"},
		{PID: 21721, RSS: 100 << 20, ProjectDir: "/Users/x/work/itaskid", StartedAt: time.Now().Add(-2 * time.Hour), LatestActivity: "回复 · 11:58:00"},
	}
	m.projects = []collect.ProjectDisk{
		{Name: "-Users-x-work-zelig", Sessions: 42, Bytes: 476 << 20, LastActive: time.Now()},
		{Name: "-Users-x-work-itaskid", Sessions: 17, Bytes: 80 << 20, LastActive: time.Now()},
	}
	m.totalDisk = 556 << 20
	return m
}

func TestUpdateKeys(t *testing.T) {
	m := testModel()
	down := func(m Model) Model {
		up, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		return up.(Model)
	}

	m2 := down(down(down(m))) // 只有 2 行,第 3 次应停在最后一行
	if m2.cur != 1 {
		t.Errorf("down 溢出: cur = %d, want 1", m2.cur)
	}

	up1, _ := m2.Update(tea.KeyMsg{Type: tea.KeyUp})
	if up1.(Model).cur != 0 {
		t.Errorf("up: cur = %d, want 0", up1.(Model).cur)
	}

	tab2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if tm := tab2.(Model); tm.tab != tabDisk || tm.cur != 0 {
		t.Errorf("键 2 应切到磁盘 tab 并复位 cur,got tab=%d cur=%d", tm.tab, tm.cur)
	}

	quit, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !quit.(Model).quitting || cmd == nil {
		t.Error("键 q 应退出")
	}
}

func TestViewSmoke(t *testing.T) {
	m := testModel()
	v := m.View()
	for _, want := range []string{"1423", "zelig", "200MB", "Bash · 12:05:00", "[1] 运行中", "q 退出"} {
		if !strings.Contains(v, want) {
			t.Errorf("View() 应包含 %q\n实际:\n%s", want, v)
		}
	}

	m.tab = tabDisk
	v = m.View()
	for _, want := range []string{"-Users-x-work-zelig", "42", "476MB", "合计", "556MB"} {
		if !strings.Contains(v, want) {
			t.Errorf("磁盘 tab 应包含 %q\n实际:\n%s", want, v)
		}
	}
}

func TestViewEmpty(t *testing.T) {
	m := New(config.Default(), "/tmp/projects", 1)
	m.width, m.height = 120, 30
	v := m.View()
	if !strings.Contains(v, "无运行中的 Claude Code 会话") {
		t.Errorf("空进程态应有提示:\n%s", v)
	}
	m.tab = tabDisk
	if v = m.View(); !strings.Contains(v, "无会话记录") {
		t.Errorf("空磁盘态应有提示:\n%s", v)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		96:                      "96B",
		12 << 10:                "12KB",
		938 << 20:               "938MB",
		(1 << 30) + (600 << 20): "1.6GB",
	}
	for in, want := range cases {
		if got := HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
